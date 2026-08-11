// Package audit provides engine-agnostic audit event emission using CloudEvents
// as the envelope format. It defines the Auditor contract and ships backends
// (noop, stdout, log, kafka, multi) plus Kratos middleware that intercepts RPC
// calls and emits structured audit events.
//
// # Architecture
//
// The central abstraction is the Auditor interface (auditor.go):
//
//	type Auditor interface {
//	    Emit(ctx context.Context, event cloudevents.Event) error
//	}
//
// Implementations live in sub-packages:
//
//   - obs/audit/noop   — discards all events (testing / disabled mode)
//   - obs/audit/stdout — JSON-encodes events to stdout (local dev)
//   - obs/audit/log    — emits structured slog records (local dev / demos)
//   - obs/audit/kafka  — delivers events to Kafka via franz-go and CloudEvents binding
//   - obs/audit/multi  — fans out to multiple auditors
//
// # Middleware
//
// Middleware intercepts RPC calls, looks up generated AuditRules by operation,
// builds servora.audit.rpc.v1 CloudEvents events, and emits through the configured
// Auditor. Emission errors are logged and the handler result is preserved.
//
// Recommended middleware chain order:
//
//	recovery → tracing → logging → ratelimit → validate → metrics → audit.Middleware → additional middleware → handler
//
// # CloudEvents Attributes
//
// NewEvent sets CloudEvents required attributes and uses source="//app-name".
// RPC audit middleware sets subject to the transport operation and adds
// errormessage for handler errors. Other extensions are producer-owned, while
// backend-only fields such as partitionkey stay private to their backend package.
package audit
