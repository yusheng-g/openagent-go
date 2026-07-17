package acp

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
)

type acpCompatReader struct {
	raw        io.Reader
	writer     io.Writer
	rawBuf     []byte
	out        bytes.Buffer
	defaultCwd string
}

func newACPCompatReader(rawReader io.Reader, writer io.Writer) *acpCompatReader {
	return &acpCompatReader{
		raw:        rawReader,
		writer:     writer,
		defaultCwd: "/",
	}
}

func (r *acpCompatReader) Read(p []byte) (int, error) {
	if r.out.Len() > 0 {
		return r.out.Read(p)
	}

	for {
		line, err := r.readLine()
		if err != nil {
			return 0, err
		}
		if len(line) == 0 {
			continue
		}

		result := r.processLine(line)
		if result == nil {
			continue
		}

		r.out.Write(result)
		r.out.WriteByte('\n')
		return r.out.Read(p)
	}
}

func (r *acpCompatReader) readLine() ([]byte, error) {
	for {
		if idx := bytes.IndexByte(r.rawBuf, '\n'); idx != -1 {
			line := r.rawBuf[:idx]
			r.rawBuf = r.rawBuf[idx+1:]
			return line, nil
		}
		tmp := make([]byte, 4096)
		n, err := r.raw.Read(tmp)
		if n > 0 {
			r.rawBuf = append(r.rawBuf, tmp[:n]...)
		}
		if err != nil {
			if len(r.rawBuf) > 0 {
				line := r.rawBuf
				r.rawBuf = nil
				return line, nil
			}
			return nil, err
		}
	}
}

func (r *acpCompatReader) processLine(line []byte) []byte {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return line
	}

	methodRaw, hasMethod := raw["method"]
	if !hasMethod {
		return line
	}

	var method string
	json.Unmarshal(methodRaw, &method)

	switch method {
	case "session/setup":
		r.handleSessionSetup(raw)
		return nil

	case "session/resume", "session/load":
		return r.injectCwd(raw, line)
	}

	return line
}

func (r *acpCompatReader) handleSessionSetup(raw map[string]json.RawMessage) {
	idRaw, hasID := raw["id"]
	if !hasID {
		return
	}

	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      json.RawMessage(idRaw),
		"result": map[string]interface{}{
			"configOptions": []interface{}{},
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("acp compat: marshal session/setup response: %v", err)
		return
	}
	if _, err := r.writer.Write(append(data, '\n')); err != nil {
		log.Printf("acp compat: write session/setup response: %v", err)
	}
}

func (r *acpCompatReader) injectCwd(raw map[string]json.RawMessage, originalLine []byte) []byte {
	paramsRaw, hasParams := raw["params"]
	if !hasParams {
		return originalLine
	}

	var params map[string]interface{}
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		return originalLine
	}

	cwd, exists := params["cwd"]
	if exists {
		if cwdStr, ok := cwd.(string); ok && cwdStr != "" {
			return originalLine
		}
	}
	params["cwd"] = r.defaultCwd

	newParams, err := json.Marshal(params)
	if err != nil {
		return originalLine
	}
	raw["params"] = newParams

	newLine, err := json.Marshal(raw)
	if err != nil {
		return originalLine
	}
	return newLine
}

// Compile-time check.
var _ io.Reader = (*acpCompatReader)(nil)
