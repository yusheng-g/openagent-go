// Browser Agent example — drives a real browser via Playwright MCP.
//
// The agent receives browser_navigate, browser_click, browser_screenshot,
// browser_snapshot (accessibility tree), and other Playwright tools
// automatically through the MCP protocol. Zero openagent-go framework changes.
//
// Prerequisites:
//
//	npx @playwright/mcp@latest --help   # install once
//	go run ./examples/browser-agent/
//
//	go run ./examples/browser-agent/
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/mcp"
	"github.com/yusheng-g/openagent-go/model/openai"
)

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAGENT_API_KEY not set")
		os.Exit(1)
	}
	if modelID == "" {
		modelID = "deepseek-v4-flash"
	}
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	model := openai.New(apiKey, modelID, baseURL).WithContextWindow(128_000)

	// ── Start Playwright MCP server ──
	// The MCP server exposes browser_navigate, browser_click, browser_screenshot,
	// browser_snapshot, browser_type, browser_take_screenshot, etc.
	fmt.Println("Starting Playwright MCP server...")
	client := mcp.NewClient("browser-agent", "1.0.0")
	session, err := client.ConnectStdio(ctx, "npx", "@playwright/mcp@latest")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start Playwright MCP: %v\n", err)
		fmt.Fprintln(os.Stderr, "\nInstall it first:  npx @playwright/mcp@latest --help")
		os.Exit(1)
	}
	defer session.Close()

	tools, err := session.Tools(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list MCP tools: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d browser tools via MCP:\n", len(tools))
	for _, t := range tools {
		fmt.Printf("  - %s\n", t.Definition().Name)
	}

	// ── Build agent ──
	agent := openagent.NewAgent("browser-agent",
		openagent.WithModel(model),
		openagent.WithTools(tools...),
		openagent.WithSystemPrompts(`You control a web browser using Playwright tools. Use the available browser tools to complete tasks.

## How to fill in forms
1. browser_navigate(url) — go to the login page
2. browser_snapshot() — see all input fields and buttons (each has a "ref" like "e15", "e23")
3. browser_type(element="username field description", ref="eXX", text="the value") — type into an input field. Use the EXACT ref from the snapshot.
4. browser_type(element="password field description", ref="eYY", text="the value") — type into password field
5. browser_click(element="Login button", ref="eZZ") — click the login button
6. browser_snapshot() — confirm you're logged in or see the error message

## Key rules
- NEVER guess a ref — ALWAYS browser_snapshot() first to get the current refs.
- After typing each field, take another snapshot if the page changes.
- If you encounter SMS/phone verification (MFA), report it and STOP — you cannot receive SMS codes. Describe what you saw up to that point.
- If you see a CAPTCHA, tell the user — you cannot solve CAPTCHAs.
- If login fails, report the exact error message shown on the page.
- browser_console_messages() helps debug JavaScript errors.`),
		openagent.WithMaxTurns(15),
	)

	sess := openagent.Session{
		ID: "browser-demo", UserID: "user-1",

		CreatedAt: time.Now(),
	}

	// ── Run ──
	hwUser := os.Getenv("HW_USERNAME")
	hwPass := os.Getenv("HW_PASSWORD")

	task := os.Getenv("TASK")
	if task == "" {
		if hwUser == "" || hwPass == "" {
			task = "Go to https://www.huaweicloud.com, look at the homepage content, and tell me what products and services are featured."
		} else {
			task = fmt.Sprintf(
				"Log into HuaweiCloud console. Steps: "+
					"1. browser_navigate to https://auth.huaweicloud.com/authui/login. "+
					"2. browser_snapshot to find the fields (each has a ref like e15). "+
					"3. browser_type username %q. 4. browser_type password %q. 5. browser_click login. "+
					"6. browser_snapshot. If SMS/mobile verification appears, STOP and report it. "+
					"7. If logged in, tell me about any announcements or promotions visible.",
				hwUser, hwPass,
			)
		}
	}
	fmt.Printf("\nTask: %s\n\n", task)

	result, err := agent.Run(ctx, sess, openagent.UserMessage(task))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n── Agent response ──\n%s\n", result.FinalOutput)
	fmt.Printf("Tokens: %d prompt + %d completion = %d total, %d turns\n",
		result.Usage.PromptTokens, result.Usage.CompletionTokens,
		result.Usage.TotalTokens, result.TurnCount)
}
