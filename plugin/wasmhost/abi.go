package wasmhost

import (
	"context"

	"github.com/tetratelabs/wazero/api"
)

// Pack encodes (ptr, length) into a single u64 — high 32 bits = ptr, low 32 bits = len.
// This is the WASM ABI convention used by all plugin exports.
func Pack(ptr, length uint32) uint64 {
	return (uint64(ptr) << 32) | uint64(length)
}

// Unpack decodes a u64 into (ptr, length).
func Unpack(packed uint64) (ptr, length uint32) {
	return uint32(packed >> 32), uint32(packed & 0xFFFF_FFFF)
}

// ReadPacked reads bytes from guest memory at the (ptr, length) location
// encoded as a packed u64. Returns nil if the read fails or the pointer is
// zero.
func ReadPacked(mod api.Module, packed uint64) []byte {
	ptr, length := Unpack(packed)
	if ptr == 0 && length == 0 {
		return nil
	}
	data, ok := mod.Memory().Read(ptr, length)
	if !ok {
		return nil
	}
	out := make([]byte, length)
	copy(out, data)
	return out
}

// ReadString reads N bytes from guest memory at ptr, returning a Go string.
func ReadString(mod api.Module, ptr, length uint32) string {
	if ptr == 0 && length == 0 {
		return ""
	}
	data, ok := mod.Memory().Read(ptr, length)
	if !ok {
		return ""
	}
	return string(data)
}

// WriteString writes data into guest memory via the guest's alloc function.
// Caller provides the context for the alloc call.
func WriteString(ctx context.Context, mod api.Module, data []byte) uint64 {
	if len(data) == 0 {
		return 0
	}
	allocFn := mod.ExportedFunction("alloc")
	if allocFn == nil {
		return 0
	}
	results, err := allocFn.Call(ctx, uint64(len(data)))
	if err != nil || len(results) == 0 {
		return 0
	}
	ptr := uint32(results[0])
	mod.Memory().Write(ptr, data)
	return Pack(ptr, uint32(len(data)))
}
