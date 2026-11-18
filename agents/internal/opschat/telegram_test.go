package opschat

import (
	"fmt"
	"testing"
)

func TestTelegramMigrateTo(t *testing.T) {
	raw := []byte(`{"ok":false,"error_code":400,"description":"Bad Request: group chat was upgraded to a supergroup chat","parameters":{"migrate_to_chat_id":-1003968683203}}`)
	err := telegramAPIError{Method: "sendMessage", Status: 400, Body: string(raw), MigrateTo: migrateToChatID(raw)}
	if TelegramMigrateTo(err) != "-1003968683203" {
		t.Fatal(TelegramMigrateTo(err))
	}
	if TelegramMigrateTo(fmt.Errorf("nope")) != "" {
		t.Fatal("plain")
	}
}
