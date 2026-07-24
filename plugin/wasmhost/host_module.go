package wasmhost

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// RegisterHostModule instantiates the "host" wazero module with the
// following exports visible to WASM plugins:
//
//	keyring_get(service_ptr, service_len, key_ptr, key_len) → packed(json)
//	keyring_set(service_ptr, service_len, key_ptr, key_len, val_ptr, val_len) → packed(json)
//	keyring_delete(service_ptr, service_len, key_ptr, key_len) → packed(json)
//	http_request(method_ptr, method_len, url_ptr, url_len,
//	             headers_ptr, headers_len, body_ptr, body_len) → packed(json)
//	fs_read(path_ptr, path_len) → packed(json)   // {"data":"<base64>","error":""}
//	fs_write(path_ptr, path_len, data_ptr, data_len) → packed(json)
//	fs_readdir(path_ptr, path_len) → packed(json) // {"entries":[{"name":"...","is_dir":true},...],"error":""}
//	log_info / log_warn / log_error(msg_ptr, msg_len) → void
//	utc_now() → uint64 (nanoseconds)
//
// All functions that can fail return packed JSON with an "error" field.
// Empty error string = success.

func (h *HostAPI) RegisterHostModule(ctx context.Context, rt wazero.Runtime) error {
	read := func(mod api.Module, ptr, length uint32) string {
		return ReadString(mod, ptr, length)
	}
	write := func(ctx context.Context, mod api.Module, data []byte) uint64 {
		return WriteString(ctx, mod, data)
	}
	writeJSON := func(ctx context.Context, mod api.Module, v any) uint64 {
		b, _ := json.Marshal(v)
		return write(ctx, mod, b)
	}

	_, err := rt.NewHostModuleBuilder("host").

		// ── keyring_get → {"value": "...", "error": ""} ──
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, svcPtr, svcLen, keyPtr, keyLen uint32) uint64 {
			if h.Keyring == nil {
				return writeJSON(ctx, mod, map[string]string{"error": "keyring not available"})
			}
			svc := read(mod, svcPtr, svcLen)
			key := read(mod, keyPtr, keyLen)
			v, err := h.Keyring.Get(svc, key)
			if err != nil {
				return writeJSON(ctx, mod, map[string]string{"error": err.Error()})
			}
			return writeJSON(ctx, mod, map[string]string{"value": v})
		}).
		Export("keyring_get").

		// ── keyring_set → {"error": ""} ──
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, svcPtr, svcLen, keyPtr, keyLen, valPtr, valLen uint32) uint64 {
			if h.Keyring == nil {
				return writeJSON(ctx, mod, map[string]string{"error": "keyring not available"})
			}
			svc := read(mod, svcPtr, svcLen)
			key := read(mod, keyPtr, keyLen)
			val := read(mod, valPtr, valLen)
			if err := h.Keyring.Set(svc, key, val); err != nil {
				return writeJSON(ctx, mod, map[string]string{"error": err.Error()})
			}
			return writeJSON(ctx, mod, map[string]string{})
		}).
		Export("keyring_set").

		// ── keyring_delete → {"error": ""} ──
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, svcPtr, svcLen, keyPtr, keyLen uint32) uint64 {
			if h.Keyring == nil {
				return writeJSON(ctx, mod, map[string]string{"error": "keyring not available"})
			}
			svc := read(mod, svcPtr, svcLen)
			key := read(mod, keyPtr, keyLen)
			if err := h.Keyring.Delete(svc, key); err != nil {
				return writeJSON(ctx, mod, map[string]string{"error": err.Error()})
			}
			return writeJSON(ctx, mod, map[string]string{})
		}).
		Export("keyring_delete").

		// ── http_request → {"status": 200, "body": "...", "error": ""} ──
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module,
			methodPtr, methodLen uint32,
			urlPtr, urlLen uint32,
			headersPtr, headersLen uint32,
			bodyPtr, bodyLen uint32) uint64 {

			if h.HTTP == nil {
				return writeJSON(ctx, mod, map[string]string{"error": "http not available"})
			}

			method := read(mod, methodPtr, methodLen)
			url := read(mod, urlPtr, urlLen)
			headersRaw := read(mod, headersPtr, headersLen)
			bodyRaw := read(mod, bodyPtr, bodyLen)

			var headers map[string]string
			if headersRaw != "" {
				json.Unmarshal([]byte(headersRaw), &headers)
			}
			status, respBody, err := h.HTTP.Do(method, url, headers, []byte(bodyRaw))

			result := struct {
				Status int    `json:"status"`
				Body   string `json:"body"`
				Error  string `json:"error,omitempty"`
			}{Status: status, Body: string(respBody)}
			if err != nil {
				result.Error = err.Error()
			}
			return writeJSON(ctx, mod, result)
		}).
		Export("http_request").

		// ── fs_read → {"data": "<base64>", "error": ""} ──
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, pathPtr, pathLen uint32) uint64 {
			if h.FS == nil {
				return writeJSON(ctx, mod, map[string]string{"error": "filesystem not available"})
			}
			path := read(mod, pathPtr, pathLen)
			data, err := h.FS.ReadFile(path)
			if err != nil {
				return writeJSON(ctx, mod, map[string]string{"error": err.Error()})
			}
			return writeJSON(ctx, mod, map[string]string{"data": base64.StdEncoding.EncodeToString(data)})
		}).
		Export("fs_read").

		// ── fs_write → {"error": ""} ──
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, pathPtr, pathLen, dataPtr, dataLen uint32) uint64 {
			if h.FS == nil {
				return writeJSON(ctx, mod, map[string]string{"error": "filesystem not available"})
			}
			path := read(mod, pathPtr, pathLen)
			raw, ok := mod.Memory().Read(dataPtr, dataLen)
			if !ok {
				return writeJSON(ctx, mod, map[string]string{"error": "read guest memory out of bounds"})
			}
			if err := h.FS.WriteFile(path, raw); err != nil {
				return writeJSON(ctx, mod, map[string]string{"error": err.Error()})
			}
			return writeJSON(ctx, mod, map[string]string{})
		}).
		Export("fs_write").

		// ── fs_readdir → {"entries": [{"name":"...","is_dir":true},...], "error": ""} ──
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, pathPtr, pathLen uint32) uint64 {
			if h.FS == nil {
				return writeJSON(ctx, mod, map[string]string{"error": "filesystem not available"})
			}
			path := read(mod, pathPtr, pathLen)
			entries, err := h.FS.ReadDir(path)
			if err != nil {
				return writeJSON(ctx, mod, map[string]string{"error": err.Error()})
			}
			type dirEntry struct {
				Name  string `json:"name"`
				IsDir bool   `json:"is_dir"`
			}
			out := make([]dirEntry, len(entries))
			for i, e := range entries {
				out[i] = dirEntry{Name: e.Name(), IsDir: e.IsDir()}
			}
			return writeJSON(ctx, mod, map[string]any{"entries": out})
		}).
		Export("fs_readdir").

		// ── log_info / log_warn / log_error → void ──
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, msgPtr uint32, msgLen uint32) {
			msg := read(mod, msgPtr, msgLen)
			if h.Logger != nil {
				h.Logger.Info(msg)
			}
		}).
		Export("log_info").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, msgPtr uint32, msgLen uint32) {
			msg := read(mod, msgPtr, msgLen)
			if h.Logger != nil {
				h.Logger.Warn(msg)
			}
		}).
		Export("log_warn").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, msgPtr uint32, msgLen uint32) {
			msg := read(mod, msgPtr, msgLen)
			if h.Logger != nil {
				h.Logger.Error(msg)
			}
		}).
		Export("log_error").

		// ── utc_now → uint64 ──
		NewFunctionBuilder().
		WithFunc(func(_ context.Context, _ api.Module) uint64 {
			return uint64(time.Now().UnixNano())
		}).
		Export("utc_now").

		// ── runtime_* → {"value":"...","error":""} ──
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module) uint64 {
			return h.runtimeGet(ctx, mod, RuntimeKeySessionID)
		}).
		Export("runtime_session_id").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module) uint64 {
			return h.runtimeGet(ctx, mod, RuntimeKeyUserID)
		}).
		Export("runtime_user_id").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module) uint64 {
			return h.runtimeGet(ctx, mod, RuntimeKeyTurnCount)
		}).
		Export("runtime_turn_count").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module) uint64 {
			return h.runtimeGet(ctx, mod, RuntimeKeyModelID)
		}).
		Export("runtime_model_id").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module) uint64 {
			return h.runtimeGet(ctx, mod, RuntimeKeyProvider)
		}).
		Export("runtime_provider").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, keyPtr, keyLen uint32) uint64 {
			key := read(mod, keyPtr, keyLen)
			return h.runtimeGet(ctx, mod, RuntimeKeyMetadataPrefix+key)
		}).
		Export("runtime_get_metadata").

		// ── runtime_set_* → {"error":""} ──
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, keyPtr, keyLen, valPtr, valLen uint32) uint64 {
			key := read(mod, keyPtr, keyLen)
			val := read(mod, valPtr, valLen)
			return h.runtimeSet(ctx, mod, RuntimeKeyMetadataPrefix+key, val)
		}).
		Export("runtime_set_metadata").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, valPtr, valLen uint32) uint64 {
			return h.runtimeSetModelConfig(ctx, mod, read(mod, valPtr, valLen))
		}).
		Export("runtime_set_model_config").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, valPtr, valLen uint32) uint64 {
			return h.runtimeSet(ctx, mod, "system_prompts", read(mod, valPtr, valLen))
		}).
		Export("runtime_set_system_prompts").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, valPtr, valLen uint32) uint64 {
			return h.runtimeSet(ctx, mod, "max_turns", read(mod, valPtr, valLen))
		}).
		Export("runtime_set_max_turns").
		Instantiate(ctx)
	return err
}

// runtimeGet reads a value from the runtime. Returns {"value":"...","error":""}.
func (h *HostAPI) runtimeGet(ctx context.Context, mod api.Module, key string) uint64 {
	rt := AgentRuntimeFromContext(ctx)
	if rt == nil {
		b, _ := json.Marshal(map[string]string{"error": "runtime not available"})
		return WriteString(ctx, mod, b)
	}
	v, ok := rt.Get(key)
	if !ok {
		b, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("key %q not found", key)})
		return WriteString(ctx, mod, b)
	}
	b, _ := json.Marshal(map[string]string{"value": v})
	return WriteString(ctx, mod, b)
}

// runtimeSet writes a value to the runtime. Returns {"error":""}.
func (h *HostAPI) runtimeSet(ctx context.Context, mod api.Module, key, value string) uint64 {
	rt := AgentRuntimeFromContext(ctx)
	if rt == nil {
		b, _ := json.Marshal(map[string]string{"error": "runtime not available"})
		return WriteString(ctx, mod, b)
	}
	if err := rt.Set(key, value); err != nil {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		return WriteString(ctx, mod, b)
	}
	b, _ := json.Marshal(map[string]string{})
	return WriteString(ctx, mod, b)
}

// runtimeSetModelConfig parses a JSON model_config and calls rt.SetModel.
// Input: {"provider":"deepseek","model_id":"v4","api_key":"sk-...","base_url":"https://..."}
// api_key and base_url are optional — empty values leave the existing ones unchanged.
func (h *HostAPI) runtimeSetModelConfig(ctx context.Context, mod api.Module, raw string) uint64 {
	rt := AgentRuntimeFromContext(ctx)
	if rt == nil {
		b, _ := json.Marshal(map[string]string{"error": "runtime not available"})
		return WriteString(ctx, mod, b)
	}
	if rt.SetModel == nil {
		b, _ := json.Marshal(map[string]string{"error": "SetModel not configured"})
		return WriteString(ctx, mod, b)
	}
	var mc struct {
		Provider string `json:"provider"`
		ModelID  string `json:"model_id"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
	}
	if err := json.Unmarshal([]byte(raw), &mc); err != nil {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		return WriteString(ctx, mod, b)
	}
	if mc.Provider == "" || mc.ModelID == "" {
		b, _ := json.Marshal(map[string]string{"error": "provider and model_id are required"})
		return WriteString(ctx, mod, b)
	}
	rt.SetModel(mc.Provider, mc.ModelID, mc.APIKey, mc.BaseURL)
	b, _ := json.Marshal(map[string]string{})
	return WriteString(ctx, mod, b)
}
