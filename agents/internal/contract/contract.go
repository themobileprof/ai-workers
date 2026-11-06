// Package contract is the n8n ↔ Go JSON envelope.
// Every department accepts Request and returns Response. context_data is always an object.
package contract

import (
	"encoding/json"
	"strings"
)

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
	ChatID         string         `json:"chat_id,omitempty"`
}

// AttachChatID copies context_data.chat_id onto the response so n8n can Send
// after the HTTP node replaces the Prepare item.
func AttachChatID(resp *Response, req Request) {
	if resp == nil || resp.ChatID != "" {
		return
	}
	raw, _ := ContextObject(req.ContextData)["chat_id"].(string)
	resp.ChatID = strings.TrimSpace(raw)
	if resp.ChatID != "" {
		if resp.StructuredData == nil {
			resp.StructuredData = map[string]any{}
		}
		if _, ok := resp.StructuredData["chat_id"]; !ok {
			resp.StructuredData["chat_id"] = resp.ChatID
		}
	}
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
