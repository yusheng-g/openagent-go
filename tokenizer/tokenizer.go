// Package tokenizer provides model-aware token counting using tiktoken.
//
// It maps model IDs to the correct BPE encoding (o200k_base for GPT-4o,
// cl100k_base for GPT-4/3.5, etc.) and caches encoder instances. For
// non-OpenAI models whose tokenizer is unknown, it falls back to cl100k_base
// which provides a reasonable approximation for English text.
//
// This package only works with strings. Message-level counting (including
// tool calls and formatting overhead) lives in the root openagent package.
package tokenizer

import (
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

var (
	mu    sync.RWMutex
	cache = make(map[string]*tiktoken.Tiktoken)
)

// ForModel returns a Tiktoken instance for the given model ID.
// Falls back to cl100k_base when the model's tokenizer is unknown.
func ForModel(modelID string) *tiktoken.Tiktoken {
	// Fast path: cache hit.
	mu.RLock()
	tke, ok := cache[modelID]
	mu.RUnlock()
	if ok {
		return tke
	}

	mu.Lock()
	defer mu.Unlock()

	// Double-check (another goroutine may have populated it).
	if tke, ok = cache[modelID]; ok {
		return tke
	}

	// Try model-specific encoding first.
	tke, err := tiktoken.EncodingForModel(modelID)
	if err != nil {
		// Unknown model — fall back to cl100k_base (reasonable for most
		// modern models that use a GPT-4-like tokenizer).
		tke, err = tiktoken.GetEncoding(tiktoken.MODEL_CL100K_BASE)
		if err != nil {
			// Absolute last resort: o200k_base is always available.
			tke, _ = tiktoken.GetEncoding(tiktoken.MODEL_O200K_BASE)
		}
	}

	cache[modelID] = tke
	return tke
}

// Count returns the token count for text using the tokenizer appropriate
// for the given model ID. Returns 0 for empty strings.
//
// If the tokenizer cannot be loaded (e.g. no network for downloading encoder
// files on first use), it falls back to a heuristic (~4 chars per token for
// ASCII, ~1.5 for CJK). This is safe but less accurate.
func Count(modelID, text string) (n int) {
	if text == "" {
		return 0
	}

	// Panic recovery: tiktoken panics on nil encoder, and encoder loading
	// fails when the network is unavailable. Fall back to heuristic.
	defer func() {
		if r := recover(); r != nil {
			n = heuristicCount(text)
		}
	}()

	tke := ForModel(modelID)
	if tke == nil {
		return heuristicCount(text)
	}
	return len(tke.EncodeOrdinary(text))
}

// IsKnownModel reports whether tiktoken has a model-specific encoding for
// the given model ID. When false, ForModel falls back to cl100k_base and
// token estimates may diverge significantly from the model's actual
// tokenizer (especially for Chinese models like GLM, Qwen, etc.). Callers
// use this to apply larger safety margins for unknown tokenizers.
func IsKnownModel(modelID string) bool {
	_, err := tiktoken.EncodingForModel(modelID)
	return err == nil
}

// heuristicCount is a fast fallback (~4 chars/token, conservative for CJK).
func heuristicCount(text string) int {
	n := 0
	for _, c := range text {
		if c > 127 {
			n += 2 // CJK and other multibyte: ~1.5-2 chars/token
		} else {
			n += 1
		}
	}
	return n/4 + 1
}
