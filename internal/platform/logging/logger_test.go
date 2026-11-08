package logging_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"araya-business-hub/be/internal/platform/logging"
)

func TestSensitiveRedaction(t *testing.T) {
	buf := new(bytes.Buffer)
	logging.SetOutput(buf)

	meta := map[string]any{
		"user_id":     10,
		"password":    "secretP@ss123",
		"jwt_token":   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.dummy.sig",
		"service_key": "sec_srv_9999",
		"email":       "user@example.com",
	}

	logging.Info("User attempted login", meta)

	var entry logging.LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to decode json log: %v", err)
	}

	if entry.Metadata["password"] != "[REDACTED]" {
		t.Errorf("expected password to be redacted, got %v", entry.Metadata["password"])
	}
	if entry.Metadata["jwt_token"] != "[REDACTED]" {
		t.Errorf("expected jwt_token to be redacted, got %v", entry.Metadata["jwt_token"])
	}
	if entry.Metadata["service_key"] != "[REDACTED]" {
		t.Errorf("expected service_key to be redacted, got %v", entry.Metadata["service_key"])
	}
	if entry.Metadata["email"] != "user@example.com" {
		t.Errorf("expected email to be preserved, got %v", entry.Metadata["email"])
	}
}
