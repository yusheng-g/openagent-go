package http

import (
	"context"
	"log"
	"os"

	"github.com/yusheng-g/openagent-go/cmd/cli/plugin"
	nativeplugin "github.com/yusheng-g/openagent-go/cmd/cli/plugin/native"
)

type ModelResource struct {
	ApiKey  string `json:"apiKey"`
	BaseUrl string `json:"baseUrl"`
	ModelId string `json:"modelId"`
	Label   string `json:"label"`
}

var availableModels []ModelResource

func loadModels(mgr *plugin.Manager, nativeMgr *nativeplugin.Manager) []ModelResource {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")

	if apiKey == "" || modelID == "" {
		log.Fatal("set OPENAGENT_API_KEY and OPENAGENT_MODEL")
	}

	availableModels = append(availableModels, ModelResource{
		ApiKey:  apiKey,
		BaseUrl: baseURL,
		ModelId: "deepseek-v4-flash",
		Label:   "deepseek",
	})
	availableModels = append(availableModels, ModelResource{
		ApiKey:  apiKey,
		BaseUrl: baseURL,
		ModelId: "deepseek-v4-pro",
		Label:   "deepseek",
	})

	// ── Model provider plugins (WASM) ──
	ctx := context.Background()
	for _, mp := range mgr.ModelProviders() {
		entries, err := mp.Load(ctx)
		if err != nil {
			log.Printf("WARNING: model provider plugin %q: %v", mp.Name(), err)
			continue
		}
		for _, e := range entries {
			availableModels = append(availableModels, ModelResource{
				ApiKey:  e.ApiKey,
				BaseUrl: e.BaseUrl,
				ModelId: e.ModelId,
				Label:   e.Label,
			})
		}
	}

	// ── Model provider plugins (native) ──
	for _, mp := range nativeMgr.ModelProviders() {
		entries, err := mp.Load(ctx)
		if err != nil {
			log.Printf("WARNING: native model provider plugin %q: %v", mp.Name(), err)
			continue
		}
		for _, e := range entries {
			availableModels = append(availableModels, ModelResource{
				ApiKey:  e.ApiKey,
				BaseUrl: e.BaseUrl,
				ModelId: e.ModelId,
				Label:   e.Label,
			})
		}
	}

	return availableModels
}
