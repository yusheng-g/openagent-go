package rest

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/eventbus"
)

// sessionEntry is implemented by every session state type.
// sessionManager accesses shared fields through this — it never
// knows the concrete type.
type sessionEntry interface {
	sessionInfo() *SessionInfo
}

// sessionHooks parameterises sessionManager by entry type E.
// Each Handler supplies its own factory and callbacks.
type sessionHooks[E sessionEntry] struct {
	kind       string
	newEntry   func(info SessionInfo) E
	fillDetail func(e E, detail *SessionDetail)
	onDelete   func(e E)
}

// ── sessionManager ──

// sessionManager handles session CRUD, store-backed restores,
// message listing, and bus subscriptions for a single mode
// ("single", "team", or "plan"). All three Handlers own one.
type sessionManager[E sessionEntry] struct {
	store  SessionStore
	memory openagent.Memory
	bus    *eventbus.Bus[SSEEvent]
	hooks  sessionHooks[E]

	mu      sync.RWMutex
	entries map[string]E
}

func newSessionManager[E sessionEntry](
	store SessionStore,
	memory openagent.Memory,
	bus *eventbus.Bus[SSEEvent],
	hooks sessionHooks[E],
) *sessionManager[E] {
	return &sessionManager[E]{
		store:   store,
		memory:  memory,
		bus:     bus,
		hooks:   hooks,
		entries: make(map[string]E),
	}
}

func (sm *sessionManager[E]) SetStore(s SessionStore)      { sm.store = s }
func (sm *sessionManager[E]) Bus() *eventbus.Bus[SSEEvent]  { return sm.bus }
func (sm *sessionManager[E]) Memory() openagent.Memory      { return sm.memory }
func (sm *sessionManager[E]) Store() SessionStore            { return sm.store }

// Exists reports whether the session is in the in-memory map.
func (sm *sessionManager[E]) Exists(id string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, ok := sm.entries[id]
	return ok
}

// withMeta runs fn against the entry's SessionInfo under sm.mu.RLock.
// Returns a snapshot so callers can persist after the lock is released.
func (sm *sessionManager[E]) withMeta(id string, fn func(*SessionInfo)) (*SessionInfo, bool) {
	sm.mu.RLock()
	e, ok := sm.entries[id]
	if !ok {
		sm.mu.RUnlock()
		return nil, false
	}
	fn(e.sessionInfo())
	out := *e.sessionInfo()
	sm.mu.RUnlock()
	return &out, true
}

// syncMeta persists a SessionInfo snapshot to the configured store.
// No-op when store is nil.
func (sm *sessionManager[E]) syncMeta(inf *SessionInfo) {
	if sm.store == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := sm.store.Save(ctx, *inf); err != nil {
			log.Printf("rest: failed to persist meta for session %s: %v", inf.ID, err)
		}
	}()
}

// ── HTTP handlers ──

func (sm *sessionManager[E]) create(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	id := generateID()
	now := time.Now()
	info := SessionInfo{
		ID:        id,
		Title:     req.Title,
		Kind:      sm.hooks.kind,
		ModelID:   req.ModelID,
		Provider:  req.Provider,
		CreatedAt: now,
		UpdatedAt: now,
	}

	sm.mu.Lock()
	sm.entries[id] = sm.hooks.newEntry(info)
	sm.mu.Unlock()

	sm.syncMeta(&info)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(info)
}

func (sm *sessionManager[E]) list(w http.ResponseWriter, r *http.Request) {
	if sm.store != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		list, err := sm.store.List(ctx, sm.hooks.kind)
		if err != nil {
			http.Error(w, `{"error":"failed to list sessions"}`, http.StatusInternalServerError)
			return
		}
		if list == nil {
			list = []SessionInfo{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)
		return
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	list := make([]SessionInfo, 0, len(sm.entries))
	for _, e := range sm.entries {
		list = append(list, *e.sessionInfo())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (sm *sessionManager[E]) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sm.mu.RLock()
	e, ok := sm.entries[id]
	sm.mu.RUnlock()

	if ok {
		detail := SessionDetail{SessionInfo: *e.sessionInfo()}
		if sm.hooks.fillDetail != nil {
			sm.hooks.fillDetail(e, &detail)
		}
		if sm.memory != nil {
			if n, _ := sm.memory.Count(context.Background(), id); n > 0 {
				detail.MessageCount = n
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
		return
	}

	// Not in memory — try persistent store (e.g. after restart).
	if sm.store != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		stored, err := sm.store.Get(ctx, id)
		if err != nil {
			http.Error(w, `{"error":"failed to get session"}`, http.StatusInternalServerError)
			return
		}
		if stored != nil {
			detail := SessionDetail{SessionInfo: *stored}
			if sm.memory != nil {
				if n, _ := sm.memory.Count(context.Background(), id); n > 0 {
					detail.MessageCount = n
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(detail)
			return
		}
	}

	http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
}

func (sm *sessionManager[E]) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	// Check store if session isn't in memory (restart recovery).
	if sm.store != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if stored, err := sm.store.Get(ctx, id); err == nil && stored != nil {
			sm.getOrCreate(id) // restore into memory
		}
	}

	inf, ok := sm.withMeta(id, func(inf *SessionInfo) {
		inf.Title = body.Title
		if body.Title != "" {
			inf.UpdatedAt = time.Now()
		}
	})
	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}
	sm.syncMeta(inf)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(*inf)
}

func (sm *sessionManager[E]) messages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if sm.memory == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]openagent.Message{})
		return
	}

	limit := 50
	if l, err := parseIntParam(r, "limit", 1, 200); err == nil {
		limit = l
	}
	before := 0
	if b, err := parseIntParam(r, "before", 0, 100000); err == nil {
		before = b
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	msgs, err := sm.memory.Recent(ctx, id, limit, before)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch messages"}`, http.StatusInternalServerError)
		return
	}
	if msgs == nil {
		msgs = []openagent.Message{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

func (sm *sessionManager[E]) del(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Delete from store first (best-effort) to prevent TOCTOU resurrection
	// on restart. If store is temporarily unavailable, continue anyway —
	// user intent wins over storage reliability.
	if sm.store != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := sm.store.Delete(ctx, id); err != nil {
			log.Printf("rest: failed to delete session %s from store: %v", id, err)
		}
	}

	sm.mu.Lock()
	e, ok := sm.entries[id]
	if ok && sm.hooks.onDelete != nil {
		sm.hooks.onDelete(e)
	}
	delete(sm.entries, id)
	sm.mu.Unlock()

	if sm.memory != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		_ = sm.memory.DeleteSession(ctx, id)
	}

	w.WriteHeader(http.StatusNoContent)
}

// getOrCreate returns the existing entry or creates a new one.
// On first creation it checks the persistent store to restore
// metadata from a previous run (e.g. after restart).
func (sm *sessionManager[E]) getOrCreate(id string) E {
	sm.mu.Lock()
	if e, ok := sm.entries[id]; ok {
		sm.mu.Unlock()
		return e
	}
	sm.mu.Unlock()

	// Try to restore from persistent store.
	var info SessionInfo
	restored := false
	if sm.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if stored, err := sm.store.Get(ctx, id); err == nil && stored != nil {
			info = *stored
			restored = true
		}
	}

	if !restored {
		info = SessionInfo{
			ID:        id,
			Kind:      sm.hooks.kind,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check: someone else might have created it while we were
	// querying the store.
	if e, ok := sm.entries[id]; ok {
		return e
	}

	e := sm.hooks.newEntry(info)
	sm.entries[id] = e

	if !restored && sm.store != nil {
		sm.syncMeta(&info)
	}

	return e
}
