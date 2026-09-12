package contract

import "encoding/json"

type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
)

// Request is the n8n → worker payload.
type Request struct {
	TaskDescription string          `json:"task_description"`
	ContextData     json.RawMessage `json:"context_data"`
}

// Response is the worker → n8n execution contract.
type Response struct {
	Status         Status         `json:"status"`
	OutputText     string         `json:"output_text"`
	StructuredData map[string]any `json:"structured_data"`
}

func Fail(msg string) Response {
	return Response{
		Status:         StatusFailed,
		OutputText:     msg,
		StructuredData: map[string]any{},
	}
}

func ContextObject(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return map[string]any{"_raw": string(raw)}
	}
	if obj == nil {
		return map[string]any{}
	}
	return obj
}
