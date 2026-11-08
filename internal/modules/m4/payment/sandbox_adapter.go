package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"araya-business-hub/be/internal/modules/m4/domain"
)

type SandboxAdapter struct {
	secretKey string
}

func NewSandboxAdapter(secretKey string) *SandboxAdapter {
	if secretKey == "" {
		secretKey = "sandbox-secret-key-12345"
	}
	return &SandboxAdapter{secretKey: secretKey}
}

func formatPaymentRef(invoiceID int) string {
	return fmt.Sprintf("PAY-SANDBOX-%d-%d", invoiceID, time.Now().UnixNano()%1000000)
}

func (a *SandboxAdapter) GenerateSignature(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(a.secretKey))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *SandboxAdapter) VerifySignature(payload []byte, signature string) bool {
	expected := a.GenerateSignature(payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (a *SandboxAdapter) ProcessPayment(invoiceID int, amount int64, method string, simFail bool) (*domain.Payment, error) {
	if amount <= 0 {
		return nil, errors.New("nominal pembayaran harus lebih dari 0")
	}
	switch method {
	case "sandbox_qris", "sandbox_va", "sandbox_bank_transfer":
	default:
		return nil, errors.New("metode pembayaran sandbox tidak didukung")
	}
	status := domain.PaymentSuccess
	if simFail {
		status = domain.PaymentFailed
	}
	now := time.Now().UTC()
	return &domain.Payment{
		PaymentReference: formatPaymentRef(invoiceID),
		InvoiceID:        invoiceID,
		Amount:           amount,
		Method:           method,
		Status:           status,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}
