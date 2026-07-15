package http

import (
	"log"

	"github.com/yusheng-g/openagent-go/memory/sqlite"
)

func loadMemory() *sqlite.Memory {
	// ── Memory ──
	mem, err := sqlite.New("./backend-memory.db")
	if err != nil {
		log.Fatalf("memory: %v", err)
	}
	return mem
}
