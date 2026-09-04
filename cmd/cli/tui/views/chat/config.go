package chat

import (
	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
)

// sessionConfigOption returns the session config option whose ID or Category
// matches id, or nil. Session options are addressed by ID ("model") and may
// carry a Category alias ("model"); the server uses both interchangeably, so a
// single helper centralizes the dual-match instead of repeating it per call.
func sessionConfigOption(opts []openacp.SessionConfigOption, id string) *openacp.SessionConfigOption {
	for i := range opts {
		if opts[i].ID == openacp.SessionConfigId(id) || opts[i].Category == id {
			return &opts[i]
		}
	}
	return nil
}

// sessionConfigValue returns the string current value of the config option id,
// or "" when the option is absent or its current value is not a string.
func sessionConfigValue(opts []openacp.SessionConfigOption, id string) string {
	if o := sessionConfigOption(opts, id); o != nil {
		if s, ok := o.CurrentValue.(string); ok {
			return s
		}
	}
	return ""
}

// sessionConfigValues returns the selectable values of the config option id
// (e.g. the model list), or nil when the option is absent or has no options.
func sessionConfigValues(opts []openacp.SessionConfigOption, id string) []string {
	if o := sessionConfigOption(opts, id); o != nil {
		out := make([]string, 0, len(o.Options))
		for _, v := range o.Options {
			out = append(out, v.Value)
		}
		return out
	}
	return nil
}
