package departments

import "testing"

func TestTakeImagesStripsBase64AndSniffsPNG(t *testing.T) {
	data := map[string]any{
		"channel":       "telegram",
		"image_base64":  "iVBORw0KGgo=", // not a full png; sniff will default jpeg if magic missing
		"image_mime":    "image/png",
		"photo_file_id": "abc",
		"vendor_hint":   "Apex",
	}
	imgs := TakeImages(data)
	if len(imgs) != 1 || imgs[0].MIME != "image/png" || len(imgs[0].Data) == 0 {
		t.Fatalf("imgs=%+v", imgs)
	}
	if _, ok := data["image_base64"]; ok {
		t.Fatal("image_base64 should be stripped from the text prompt context")
	}
	if data["vendor_hint"] != "Apex" {
		t.Fatalf("lost unrelated context: %+v", data)
	}
}

func TestTakeImagesEmpty(t *testing.T) {
	if imgs := TakeImages(map[string]any{"channel": "telegram"}); len(imgs) != 0 {
		t.Fatalf("unexpected %+v", imgs)
	}
}
