package testdata

import (
	"strings"
	"testing"
)

func TestRedactDoesNotMutateMessage(t *testing.T) {
	req := &LoginRequest{
		Email:    "user@example.com",
		Password: "plain-password",
		CapToken: []byte("cap-secret"),
	}

	logged := req.Redact()
	if strings.Contains(logged, "plain-password") || strings.Contains(logged, "cap-secret") {
		t.Fatalf("redacted output contains secret: %q", logged)
	}
	if req.Password != "plain-password" || string(req.CapToken) != "cap-secret" {
		t.Fatalf("Redact mutated original message: %#v", req)
	}
}

func TestRedactFieldsMutatesOnlyAnnotatedFields(t *testing.T) {
	req := &LoginRequest{
		Email:    "user@example.com",
		Password: "plain-password",
		CapToken: []byte("cap-secret"),
	}

	req.RedactFields()
	if req.Email != "user@example.com" {
		t.Fatalf("RedactFields changed unannotated email: %q", req.Email)
	}
	if req.Password != "" || len(req.CapToken) != 0 {
		t.Fatalf("RedactFields did not clear annotated fields: %#v", req)
	}
}
