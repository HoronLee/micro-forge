// Package authz provides a Kratos middleware for engine-agnostic authorization.
//
// # Engine model
//
// The Authorizer interface exposes a single Check method that accepts an
// engine-neutral CheckRequest with Subject, Action, Resource, and Attributes.
// Any authorization backend — OpenFGA, SpiceDB, Cedar, OPA, or custom —
// can implement this interface.
//
// Batch and list capabilities are available through the root BatchAuthorizer
// and Lister interfaces, which engines may implement as needed.
//
// # Subject resolution
//
// The middleware reads the standard authenticated subject through
// authn.SubjectFrom. WithSubjectFunc overrides that reader for workload
// identity, external authentication middleware, and tests.
//
// # Audit integration
//
// When WithAuditor is configured with an audit.Auditor, the middleware emits
// typed CloudEvents events for allowed, denied, and error decisions. The typed
// payload contains the authenticated subject; the CloudEvents subject identifies
// the authorization resource. Without the option, the middleware is silent.
//
// # Future: contextual tuples / attributes
//
// The CheckRequest.Attributes field (map[string]any) is reserved for
// request-level facts that participate in a decision but are not persisted:
// device trust, active session, time-of-day, request region, etc.
// Engines that support ABAC or contextual tuples can read from this field.
package authz
