// Package opschat is the office Telegram assistant: one desk per turn, then that Handle.
package opschat

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Inbound is one staff Telegram message after JSON parse.
type Inbound struct {
	ChatID         string
	FromID         string
	Text           string
	ReplyTo        string
	HasPhoto       bool
	HasDocument    bool
	PhotoFileID    string
	DocumentFileID string
	Filename       string
	FileMIME       string
	Group          bool
	FromBot        bool
}

type tgUpdate struct {
	Message *tgMessage `json:"message"`
}

type tgMessage struct {
	From           tgUser      `json:"from"`
	Chat           tgChat      `json:"chat"`
	Text           string      `json:"text"`
	Caption        string      `json:"caption"`
	Photo          []tgPhoto   `json:"photo"`
	Document       *tgDocument `json:"document"`
	ReplyToMessage *tgMessage  `json:"reply_to_message"`
}

type tgUser struct {
	ID    int64  `json:"id"`
	IsBot bool   `json:"is_bot"`
	Name  string `json:"first_name"`
}

type tgChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type tgPhoto struct {
	FileID string `json:"file_id"`
}

type tgDocument struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	Mime     string `json:"mime_type"`
}

// ParseUpdate reads a Telegram Update JSON object.
func ParseUpdate(raw []byte) (Inbound, bool) {
	var u tgUpdate
	if json.Unmarshal(raw, &u) != nil || u.Message == nil {
		return Inbound{}, false
	}
	m := u.Message
	text := strings.TrimSpace(m.Text)
	if text == "" {
		text = strings.TrimSpace(m.Caption)
	}
	in := Inbound{
		ChatID:  strconv.FormatInt(m.Chat.ID, 10),
		FromID:  strconv.FormatInt(m.From.ID, 10),
		Text:    text,
		FromBot: m.From.IsBot,
		Group:   m.Chat.Type == "group" || m.Chat.Type == "supergroup",
	}
	if m.ReplyToMessage != nil {
		in.ReplyTo = strings.TrimSpace(m.ReplyToMessage.Text)
		if in.ReplyTo == "" {
			in.ReplyTo = strings.TrimSpace(m.ReplyToMessage.Caption)
		}
	}
	if n := len(m.Photo); n > 0 {
		in.HasPhoto = true
		in.PhotoFileID = m.Photo[n-1].FileID
	}
	if m.Document != nil && m.Document.FileID != "" {
		in.HasDocument = true
		in.DocumentFileID = m.Document.FileID
		in.Filename = m.Document.FileName
		in.FileMIME = m.Document.Mime
	}
	if in.FromBot || in.ChatID == "" || in.FromID == "" {
		return in, false
	}
	if in.Text == "" && !in.HasPhoto && !in.HasDocument {
		return in, false
	}
	return in, true
}

// ClipText fits Telegram's sendMessage limit.
func ClipText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "I heard you."
	}
	if len(s) > 4000 {
		return strings.TrimSpace(s[:3990]) + "…"
	}
	return s
}
