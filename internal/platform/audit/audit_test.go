package audit_test

import (
	"context"
	"testing"

	"araya-business-hub/be/internal/platform/audit"
)

func TestAuditService(t *testing.T) {
	ctx := context.Background()
	repo := audit.NewInMemoryRepository()
	service := audit.NewService(repo)

	userID := 5
	record := audit.Record{
		UserID:     &userID,
		UserName:   "Admin Utama",
		Action:     "user.password_reset",
		EntityType: "user",
		EntityID:   "12",
		Details: map[string]any{
			"password":  "plaintextSecret",
			"target_id": 12,
		},
		IPAddress: "192.168.1.1",
		RequestID: "req-1234-uuid",
	}

	if err := service.Log(ctx, record); err != nil {
		t.Fatalf("failed to log audit: %v", err)
	}

	list, total, err := service.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to list audit records: %v", err)
	}

	if total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 record, got %d", len(list))
	}

	first := list[0]
	if first.Action != "user.password_reset" {
		t.Errorf("expected action 'user.password_reset', got %s", first.Action)
	}

	// Verify sensitive detail was automatically redacted
	if first.Details["password"] != "[REDACTED]" {
		t.Errorf("expected password in audit details to be redacted, got %v", first.Details["password"])
	}
	if first.Details["target_id"] != 12 {
		t.Errorf("expected target_id 12, got %v", first.Details["target_id"])
	}
}
