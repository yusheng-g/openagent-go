package wasmhost

import (
	"context"
	"encoding/json"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// HostAPI bundles the three host-provided capabilities available to WASM
// plugins via the "host" module.
type HostAPI struct {
	Keyring Keyring
	HTTP    HTTPClient
	Logger  Logger
}

// RegisterHostModule instantiates the "host" wazero module with the
// following exports visible to WASM plugins:
//
//	keyring_get(service_ptr, service_len, key_ptr, key_len) → packed(str)
//	keyring_set(service_ptr, service_len, key_ptr, key_len, val_ptr, val_len)
//	keyring_delete(service_ptr, service_len, key_ptr, key_len)
//	http_request(method_ptr, method_len, url_ptr, url_len,
//	             headers_ptr, headers_len, body_ptr, body_len) → packed(json)
//	log_info(msg_ptr, msg_len)
//	log_warn(msg_ptr, msg_len)
//	log_error(msg_ptr, msg_len)
//
// Caller must call this BEFORE instantiating any plugin module that
// imports from "host".
func (h *HostAPI) RegisterHostModule(ctx context.Context, rt wazero.Runtime) error {
	read := func(mod api.Module, ptr, length uint32) string {
		return ReadString(mod, ptr, length)
	}
	write := func(ctx context.Context, mod api.Module, data []byte) uint64 {
		return WriteString(ctx, mod, data)
	}

	_, err := rt.NewHostModuleBuilder("host").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, svcPtr, svcLen, keyPtr, keyLen uint32) uint64 {
			if h.Keyring == nil {
				return 0
			}
			svc := read(mod, svcPtr, svcLen)
			key := read(mod, keyPtr, keyLen)
			v, _ := h.Keyring.Get(svc, key)
			return write(ctx, mod, []byte(v))
		}).
		Export("keyring_get").

		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, svcPtr, svcLen, keyPtr, keyLen, valPtr, valLen uint32) {
			if h.Keyring == nil {
				return
			}
			svc := read(mod, svcPtr, svcLen)
			key := read(mod, keyPtr, keyLen)
			val := read(mod, valPtr, valLen)
			_ = h.Keyring.Set(svc, key, val)
		}).
		Export("keyring_set").

		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, svcPtr, svcLen, keyPtr, keyLen uint32) {
			svc := read(mod, svcPtr, svcLen)
			key := read(mod, keyPtr, keyLen)
			_ = h.Keyring.Delete(svc, key)
		}).
		Export("keyring_delete").

		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module,
			methodPtr, methodLen uint32,
			urlPtr, urlLen uint32,
			headersPtr, headersLen uint32,
			bodyPtr, bodyLen uint32) uint64 {

			method := read(mod, methodPtr, methodLen)
			url := read(mod, urlPtr, urlLen)
			headersRaw := read(mod, headersPtr, headersLen)
			bodyRaw := read(mod, bodyPtr, bodyLen)

			var headers map[string]string
			if headersRaw != "" {
				json.Unmarshal([]byte(headersRaw), &headers)
			}
			status, respBody, _ := h.HTTP.Do(method, url, headers, []byte(bodyRaw))

			var result struct {
				Status int    `json:"status"`
				Body   string `json:"body"`
			}
			result.Status = status
			result.Body = string(respBody)
			respJSON, _ := json.Marshal(result)
			return write(ctx, mod, respJSON)
		}).
		Export("http_request").

		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, msgPtr uint32, msgLen uint32) {
			msg := read(mod, msgPtr, msgLen)
			h.Logger.Info(msg)
		}).
		Export("log_info").

		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, msgPtr uint32, msgLen uint32) {
			msg := read(mod, msgPtr, msgLen)
			h.Logger.Warn(msg)
		}).
		Export("log_warn").

		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, mod api.Module, msgPtr uint32, msgLen uint32) {
			msg := read(mod, msgPtr, msgLen)
			h.Logger.Error(msg)
		}).
		Export("log_error").

		Instantiate(ctx)
	return err
}
