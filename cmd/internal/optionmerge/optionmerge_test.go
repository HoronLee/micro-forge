package optionmerge

import (
	"testing"

	auditv1 "github.com/Servora-Kit/servora/api/gen/go/servora/audit/v1"
)

func TestMergeAudit(t *testing.T) {
	tests := []struct {
		name      string
		svc       *auditv1.AuditRule
		method    *auditv1.AuditRule
		hasMethod bool
		wantOK    bool
		wantMode  auditv1.AuditMode
	}{
		{name: "both nil returns false"},
		{
			name:     "service default wins",
			svc:      &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED},
			wantOK:   true,
			wantMode: auditv1.AuditMode_AUDIT_MODE_ENABLED,
		},
		{
			name:      "method rule wins",
			svc:       &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED},
			method:    &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_DISABLED},
			hasMethod: true,
			wantOK:    true,
			wantMode:  auditv1.AuditMode_AUDIT_MODE_DISABLED,
		},
		{
			name:      "unspecified method inherits service",
			svc:       &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED},
			method:    &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_UNSPECIFIED},
			hasMethod: true,
			wantOK:    true,
			wantMode:  auditv1.AuditMode_AUDIT_MODE_ENABLED,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := Merge(test.svc, test.method, test.hasMethod)
			if ok != test.wantOK {
				t.Fatalf("Merge() ok = %v, want %v", ok, test.wantOK)
			}
			if ok && got.GetMode() != test.wantMode {
				t.Fatalf("Merge() mode = %v, want %v", got.GetMode(), test.wantMode)
			}
		})
	}
}

func TestMergeDeepClonesRule(t *testing.T) {
	original := &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED}
	merged, ok := Merge[*auditv1.AuditRule](nil, original, true)
	if !ok {
		t.Fatal("expected effective rule")
	}
	merged.Mode = auditv1.AuditMode_AUDIT_MODE_DISABLED
	if original.GetMode() != auditv1.AuditMode_AUDIT_MODE_ENABLED {
		t.Fatal("Merge mutated original rule")
	}
}
