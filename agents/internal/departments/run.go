package departments

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const jsonContract = `
Respond with a single JSON object and nothing else (no markdown fences). Shape:
{
  "status": "success" or "failed",
  "output_text": "Comprehensive Markdown detailing the work/findings.",
  "structured_data": { ... extracted or calculated fields ... },
  "task_type": "one of the allowed sub-tasks for this department"
}
If the request cannot be completed, still return JSON with status "failed" and explain in output_text.
`

func Run(ctx context.Context, c llm.Completer, systemPrompt, userPrompt string, images ...llm.Image) (contract.Response, error) {
	text, err := c.Complete(ctx, llm.Request{
		System:   strings.TrimSpace(systemPrompt) + "\n" + jsonContract,
		Messages: []llm.Message{{Role: "user", Content: userPrompt}},
		Images:   images,
	})
	if err != nil {
		return contract.Fail("LLM call failed: " + err.Error()), err
	}
	parsed, err := decodeResponse(text)
	if err != nil {
		return contract.Response{
			Status:     contract.StatusFailed,
			OutputText: "Model returned unparseable output.\n\n" + text,
			StructuredData: map[string]any{
				"parse_error": err.Error(),
			},
		}, nil
	}
	return parsed, nil
}

func UserPrompt(taskDescription string, contextData map[string]any) string {
	ctxJSON, _ := json.MarshalIndent(contextData, "", "  ")
	return fmt.Sprintf("Task description:\n%s\n\ncontext_data:\n%s\n", taskDescription, string(ctxJSON))
}

func decodeResponse(raw string) (contract.Response, error) {
	trimmed := strings.TrimSpace(raw)
	if i := strings.Index(trimmed, "{"); i >= 0 {
		if j := strings.LastIndex(trimmed, "}"); j > i {
			trimmed = trimmed[i : j+1]
		}
	}
	var envelope struct {
		Status         string         `json:"status"`
		OutputText     string         `json:"output_text"`
		StructuredData map[string]any `json:"structured_data"`
		TaskType       string         `json:"task_type"`
	}
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return contract.Response{}, err
	}
	status := contract.Status(strings.ToLower(strings.TrimSpace(envelope.Status)))
	if status != contract.StatusSuccess && status != contract.StatusFailed {
		status = contract.StatusFailed
	}
	data := envelope.StructuredData
	if data == nil {
		data = map[string]any{}
	}
	if envelope.TaskType != "" {
		if _, exists := data["task_type"]; !exists {
			data["task_type"] = envelope.TaskType
		}
	}
	return contract.Response{
		Status:         status,
		OutputText:     envelope.OutputText,
		StructuredData: data,
	}, nil
}
