package feishu

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yusheng-g/openagent-go/utils"
)

// approvalButtonRow builds the column_set element containing the three
// ACP-style approval buttons (Allow Once / Allow Always / Deny).
// The approvalID is embedded in every button's value so the card action
// callback can correlate clicks. "allow_always" remembers the decision
// for the session (same tool+args won't ask again).
func approvalButtonRow(approvalID string) map[string]any {
	btn := func(label, name, btnType string) map[string]any {
		return map[string]any{
			"tag":   "button",
			"text":  map[string]any{"tag": "plain_text", "content": label},
			"type":  btnType,
			"width": "default",
			"name":  name,
			"value": map[string]any{"approval_id": approvalID, "action": name},
		}
	}
	return map[string]any{
		"tag":                "column_set",
		"flex_mode":          "flow",
		"horizontal_spacing": "8px",
		"columns": []map[string]any{
			{"tag": "column", "width": "auto", "elements": []map[string]any{btn("Allow Once", "allow_once", "primary_filled")}},
			{"tag": "column", "width": "auto", "elements": []map[string]any{btn("Allow Always", "allow_always", "primary_filled")}},
			{"tag": "column", "width": "auto", "elements": []map[string]any{btn("Deny", "deny", "danger_filled")}},
		},
	}
}

// buildApprovalCard constructs the Feishu interactive card JSON for a tool
// call approval request. The card shows the tool name and arguments, then
// three ACP-style action buttons (Allow Once / Allow Always / Deny).
//
// approvalID is embedded in every button's "value" field so the card action
// callback can correlate the click back to the pending approval. The card
// uses schema 2.0 with no banner image (keeping it self-contained — no
// uploaded image dependency).
func buildApprovalCard(approvalID, toolName, args string) (string, error) {
	pretty := utils.PrettyJSON(args)
	body := fmt.Sprintf("**工具名称** ：%s\n**执行参数** ：%s", toolName, pretty)

	card := map[string]any{
		"schema": "2.0",
		"config": map[string]any{"update_multi": true},
		"header": map[string]any{
			"title":    map[string]any{"tag": "plain_text", "content": "需要权限"},
			"subtitle": map[string]any{"tag": "plain_text", "content": ""},
			"template": "blue",
		},
		"body": map[string]any{
			"direction": "vertical",
			"elements": []map[string]any{
				{
					"tag":     "markdown",
					"content": body,
				},
				{"tag": "hr"},
				approvalButtonRow(approvalID),
			},
		},
	}

	b, err := json.Marshal(card)
	if err != nil {
		return "", fmt.Errorf("feishu approval card marshal: %w", err)
	}
	return string(b), nil
}

// buildResolvedCard constructs the card JSON shown after the user has made a
// decision. The card body is a collapsible_panel that starts collapsed —
// the panel header shows the decision summary, and the hidden body contains
// the tool name, args, and optional reason. The user can expand to see
// details. The header colour reflects the outcome.
func buildResolvedCard(toolName, args, decision, reason string) (string, error) {
	pretty := utils.PrettyJSON(args)
	var detail strings.Builder
	fmt.Fprintf(&detail, "**工具名称** ：%s\n**执行参数** ：%s", toolName, pretty)
	if reason != "" {
		detail.WriteString("\n\n> ")
		detail.WriteString(reason)
	}

	template := "green"
	if strings.HasPrefix(decision, "❌") {
		template = "red"
	}

	summary := decision + " — " + toolName

	panelElements := []map[string]any{
		{"tag": "markdown", "content": detail.String()},
	}

	card := map[string]any{
		"schema": "2.0",
		"header": map[string]any{
			"title":    map[string]any{"tag": "plain_text", "content": "需要权限"},
			"subtitle": map[string]any{"tag": "plain_text", "content": ""},
			"template": template,
		},
		"body": map[string]any{
			"direction": "vertical",
			"elements":  []map[string]any{panel(summary, panelElements)},
		},
	}

	b, err := json.Marshal(card)
	if err != nil {
		return "", fmt.Errorf("feishu resolved card marshal: %w", err)
	}
	return string(b), nil
}
