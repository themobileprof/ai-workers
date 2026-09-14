package departments

import (
	"bytes"
	"encoding/base64"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/llm"
)

const maxImageBytes = 5 << 20

var imageContextKeys = []string{
	"image_base64", "image_mime", "image_mime_type", "has_image",
	"photo_file_id", "whatsapp_media_id", "image_id",
}

// TakeImages pulls receipt bytes out of context_data so they are not echoed
// into the text prompt. Mutates data by deleting the bulky keys.
func TakeImages(data map[string]any) []llm.Image {
	if data == nil {
		return nil
	}
	b64, mime := imageFields(data)
	for _, k := range imageContextKeys {
		delete(data, k)
	}
	if nested, ok := data["image"].(map[string]any); ok {
		if b64 == "" {
			b64, mime = imageFields(nested)
		}
		delete(data, "image")
	}
	raw, err := decodeImage(b64)
	if err != nil || len(raw) == 0 {
		return nil
	}
	if mime == "" || mime == "application/octet-stream" {
		mime = sniffMIME(raw)
	}
	return []llm.Image{{MIME: mime, Data: raw}}
}

func imageFields(m map[string]any) (b64, mime string) {
	b64 = asString(m["image_base64"])
	if b64 == "" {
		b64 = asString(m["base64"])
	}
	mime = asString(m["image_mime"])
	if mime == "" {
		mime = asString(m["image_mime_type"])
	}
	if mime == "" {
		mime = asString(m["mime"])
	}
	if mime == "" {
		mime = asString(m["mime_type"])
	}
	return b64, mime
}

func decodeImage(b64 string) ([]byte, error) {
	s := strings.TrimSpace(b64)
	if s == "" {
		return nil, nil
	}
	if i := strings.Index(s, ","); i >= 0 && strings.Contains(strings.ToLower(s[:i]), "base64") {
		s = s[i+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(s)
	}
	if err != nil {
		return nil, err
	}
	if len(raw) > maxImageBytes {
		return nil, nil
	}
	return raw, nil
}

func sniffMIME(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg"
	case bytes.HasPrefix(b, []byte{0x89, 0x50, 0x4E, 0x47}):
		return "image/png"
	case len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func asString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}
