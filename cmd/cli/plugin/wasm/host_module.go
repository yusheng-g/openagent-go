package wasm

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/yusheng-g/openagent-go/cmd/cli/plugin"
)

type hostAPI struct {
	keyring plugin.Keyring
	http    plugin.HTTPClient
	logger  plugin.Logger
}

func (h *hostAPI) registerModule(ctx context.Context, rt wazero.Runtime) error {
	readStr := func(mod api.Module, ptr, length uint32) string {
		if ptr == 0 && length == 0 {
			return ""
		}
		data, ok := mod.Memory().Read(ptr, length)
		if !ok {
			return ""
		}
		return string(data)
	}

	writeStr := func(mod api.Module, data []byte) uint64 {
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
		return uint64(ptr)<<32 | uint64(len(data))
	}

	_, err := rt.NewHostModuleBuilder("host").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, svcPtr, svcLen, keyPtr, keyLen uint32) uint64 {
			svc := readStr(mod, svcPtr, svcLen)
			key := readStr(mod, keyPtr, keyLen)
			v, _ := h.keyring.Get(svc, key)
			return writeStr(mod, []byte(v))
		}).
		Export("keyring_get").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, svcPtr, svcLen, keyPtr, keyLen, valPtr, valLen uint32) {
			svc := readStr(mod, svcPtr, svcLen)
			key := readStr(mod, keyPtr, keyLen)
			val := readStr(mod, valPtr, valLen)
			_ = h.keyring.Set(svc, key, val)
		}).
		Export("keyring_set").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, svcPtr, svcLen, keyPtr, keyLen uint32) {
			svc := readStr(mod, svcPtr, svcLen)
			key := readStr(mod, keyPtr, keyLen)
			_ = h.keyring.Delete(svc, key)
		}).
		Export("keyring_delete").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module,
			methodPtr, methodLen uint32,
			urlPtr, urlLen uint32,
			headersPtr, headersLen uint32,
			bodyPtr, bodyLen uint32) uint64 {

			method := readStr(mod, methodPtr, methodLen)
			url := readStr(mod, urlPtr, urlLen)
			headersRaw := readStr(mod, headersPtr, headersLen)
			bodyRaw := readStr(mod, bodyPtr, bodyLen)

			var headers map[string]string
			if headersRaw != "" {
				json.Unmarshal([]byte(headersRaw), &headers)
			}

			status, respBody, _ := h.http.Do(method, url, headers, []byte(bodyRaw))

			var result struct {
				Status int    `json:"status"`
				Body   string `json:"body"`
			}
			result.Status = status
			result.Body = string(respBody)
			respJSON, _ := json.Marshal(result)
			return writeStr(mod, respJSON)
		}).
		Export("http_request").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, msgPtr uint32, msgLen uint32) {
			msg := readStr(mod, msgPtr, msgLen)
			h.logger.Info(msg)
		}).
		Export("log_info").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, msgPtr uint32, msgLen uint32) {
			msg := readStr(mod, msgPtr, msgLen)
			h.logger.Warn(msg)
		}).
		Export("log_warn").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, msgPtr uint32, msgLen uint32) {
			msg := readStr(mod, msgPtr, msgLen)
			h.logger.Error(msg)
		}).
		Export("log_error").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, keyPtr, keyLen uint32) uint64 {
			key := readStr(mod, keyPtr, keyLen)
			return writeStr(mod, []byte(os.Getenv(key)))
		}).
		Export("get_env").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module) uint64 {
			ts := time.Now().UTC().Format("20060102T150405Z")
			return writeStr(mod, []byte(ts))
		}).
		Export("get_time_utc").
		Instantiate(ctx)
	return err
}
