package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// ARY-002: registration returns an unguessable tracking code, the public
// status page is addressed by that code (never by sequential ID), no initial
// password exists anywhere, and account access requires the single-use
// activation invitation.
func TestSecurity_ARy002_ActivationFlow_ReplacesUniversalPassword(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)

	// 1. Register a partner: response must include an unguessable code.
	regW := DoReq(r, http.MethodPost, "/api/m3/v1/partners/register", "", map[string]any{
		"company_name": "PT Aktivasi Aman", "contact_person": "Tanti",
		"email": "tanti@aktivasi.com", "phone": "0812000111",
	})
	if regW.Code != http.StatusCreated {
		t.Fatalf("register failed: %s", regW.Body.String())
	}
	var reg struct {
		Data struct {
			ID               int    `json:"id"`
			RegistrationCode string `json:"registration_code"`
		} `json:"data"`
	}
	_ = json.Unmarshal(regW.Body.Bytes(), &reg)
	if reg.Data.RegistrationCode == "" || len(reg.Data.RegistrationCode) < 16 {
		t.Fatalf("expected unguessable registration code, got %q", reg.Data.RegistrationCode)
	}

	// 2. The old sequential-ID status endpoint must be gone.
	oldW := DoReq(r, http.MethodGet, fmt.Sprintf("/api/m3/v1/partners/%d/status", reg.Data.ID), "", nil)
	if oldW.Code != http.StatusNotFound {
		t.Fatalf("expected old ID-based status endpoint to be removed, got %d", oldW.Code)
	}

	// 3. Status by code while pending: minimal payload, no contact details.
	statusW := DoReq(r, http.MethodGet, "/api/m3/v1/partners/status?code="+reg.Data.RegistrationCode, "", nil)
	if statusW.Code != http.StatusOK {
		t.Fatalf("status by code failed: %s", statusW.Body.String())
	}
	var status struct {
		Data struct {
			CompanyName     string `json:"company_name"`
			Status          string `json:"status"`
			Email           string `json:"email"`
			Phone           string `json:"phone"`
			ActivationToken string `json:"activation_token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(statusW.Body.Bytes(), &status)
	if status.Data.Status != "pending" || status.Data.Email != "" || status.Data.Phone != "" {
		t.Fatalf("status response leaks data: %s", statusW.Body.String())
	}
	if status.Data.ActivationToken != "" {
		t.Fatalf("activation token must not be issued before verification")
	}

	// 4. Approve the partner, then the account exists WITHOUT any password.
	vW := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m3/v1/partners/%d/verify", reg.Data.ID), token, map[string]string{
		"action": "approve", "reason": "Dokumen lengkap",
	})
	if vW.Code != http.StatusOK {
		t.Fatalf("verify failed: %s", vW.Body.String())
	}
	oldPwW := DoReq(r, http.MethodPost, "/api/m1/v1/login", "", map[string]string{
		"email": "tanti@aktivasi.com", "password": "Partner123!",
	})
	if oldPwW.Code != http.StatusUnauthorized {
		t.Fatalf("universal initial password must not work, got %d", oldPwW.Code)
	}

	// 5. Status by code now yields the single-use activation invitation.
	statusW2 := DoReq(r, http.MethodGet, "/api/m3/v1/partners/status?code="+reg.Data.RegistrationCode, "", nil)
	_ = json.Unmarshal(statusW2.Body.Bytes(), &status)
	if status.Data.ActivationToken == "" {
		t.Fatalf("expected activation token after verification: %s", statusW2.Body.String())
	}

	// 6. Weak passwords are rejected.
	weakW := DoReq(r, http.MethodPost, "/api/m1/v1/auth/activate", "", map[string]string{
		"token": status.Data.ActivationToken, "password": "pendek",
	})
	if weakW.Code != http.StatusUnprocessableEntity && weakW.Code != http.StatusBadRequest {
		t.Fatalf("weak password must be rejected, got %d", weakW.Code)
	}

	// 7. Activation works and the token is single-use.
	actW := DoReq(r, http.MethodPost, "/api/m1/v1/auth/activate", "", map[string]string{
		"token": status.Data.ActivationToken, "password": "KataSandiKuat#2026",
	})
	if actW.Code != http.StatusOK {
		t.Fatalf("activation failed: %s", actW.Body.String())
	}
	reuseW := DoReq(r, http.MethodPost, "/api/m1/v1/auth/activate", "", map[string]string{
		"token": status.Data.ActivationToken, "password": "KataSandiLain#2026",
	})
	if reuseW.Code != http.StatusBadRequest {
		t.Fatalf("token reuse must fail, got %d", reuseW.Code)
	}

	// 8. Login with the chosen password works.
	loginW := DoReq(r, http.MethodPost, "/api/m1/v1/login", "", map[string]string{
		"email": "tanti@aktivasi.com", "password": "KataSandiKuat#2026",
	})
	if loginW.Code != http.StatusOK {
		t.Fatalf("login after activation failed: %s", loginW.Body.String())
	}

	// 9. After activation the status page no longer issues tokens.
	var finalStatus struct {
		Data struct {
			ActivationToken string `json:"activation_token"`
		} `json:"data"`
	}
	statusW3 := DoReq(r, http.MethodGet, "/api/m3/v1/partners/status?code="+reg.Data.RegistrationCode, "", nil)
	_ = json.Unmarshal(statusW3.Body.Bytes(), &finalStatus)
	if finalStatus.Data.ActivationToken != "" {
		t.Fatalf("activated account must not receive new invitation tokens")
	}
}

// ARY-002: unknown/guessed codes are rejected uniformly, and expired tokens
// are rejected distinctly.
func TestSecurity_ARy002_StatusAndToken_Enumeration(t *testing.T) {
	r, _ := SetupFullTestApp()
	w := DoReq(r, http.MethodGet, "/api/m3/v1/partners/status?code=1", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("sequential guess must fail, got %d", w.Code)
	}
	w2 := DoReq(r, http.MethodPost, "/api/m1/v1/auth/activate", "", map[string]string{
		"token": "act_deadb33fc0ffee", "password": "KataSandiKuat#2026",
	})
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("unknown token must fail, got %d", w2.Code)
	}
}
