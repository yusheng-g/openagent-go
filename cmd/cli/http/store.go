package http

import (
	"log"

	"github.com/yusheng-g/openagent-go/memory/sqlite"
	sessionsqlite "github.com/yusheng-g/openagent-go/rest/sessionstore/sqlite"
)

func loadSessionStore(mem *sqlite.Memory) *sessionsqlite.Store {
	sessionStore, err := sessionsqlite.New(mem.DB())
	if err != nil {
		log.Fatalf("session store: %v", err)
	}
	return sessionStore
}
