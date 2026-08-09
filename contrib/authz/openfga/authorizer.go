// Package openfga adapts the official OpenFGA SDK client to Servora's root
// authorization capabilities.
package openfga

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/Servora-Kit/servora/security/authz"
	fgasdk "github.com/openfga/go-sdk"
	fgaclient "github.com/openfga/go-sdk/client"
)

// Option configures Authorizer construction.
type Option func(*options) error

type options struct {
	logger *slog.Logger
}

// WithLogger configures provider-local diagnostics.
func WithLogger(logger *slog.Logger) Option {
	return func(opts *options) error {
		if logger == nil {
			return fmt.Errorf("openfga authz: logger is nil")
		}
		opts.logger = logger
		return nil
	}
}

// Authorizer maps Servora authorization requests to the official OpenFGA SDK.
type Authorizer struct {
	client *fgaclient.OpenFgaClient
	logger *slog.Logger
}

var (
	_ authz.Authorizer      = (*Authorizer)(nil)
	_ authz.BatchAuthorizer = (*Authorizer)(nil)
	_ authz.Lister          = (*Authorizer)(nil)
)

// New constructs an OpenFGA authorization adapter.
func New(client *fgaclient.OpenFgaClient, opts ...Option) (*Authorizer, error) {
	if client == nil {
		return nil, fmt.Errorf("openfga authz: SDK client is nil")
	}
	var cfg options
	for index, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("openfga authz: option[%d] is nil", index)
		}
		if err := opt(&cfg); err != nil {
			return nil, fmt.Errorf("openfga authz: option[%d]: %w", index, err)
		}
	}
	return &Authorizer{client: client, logger: cfg.logger}, nil
}

// Check implements authz.Authorizer.
func (a *Authorizer) Check(ctx context.Context, req authz.CheckRequest) (bool, error) {
	if err := validateCheckRequest(req); err != nil {
		return false, err
	}
	body := fgaclient.ClientCheckRequest{
		User:     req.Subject,
		Relation: req.Action,
		Object:   req.Resource.Type + ":" + req.Resource.ID,
	}
	if req.Attributes != nil {
		body.Context = &req.Attributes
	}
	response, err := a.client.Check(ctx).Body(body).Execute()
	if err != nil {
		classified := classifyProviderError(ctx, "check", err)
		a.logProviderError(ctx, "check", classified)
		return false, classified
	}
	if response == nil || !response.HasAllowed() {
		return false, fmt.Errorf("openfga check: response is missing allowed")
	}
	return response.GetAllowed(), nil
}

// BatchCheck implements authz.BatchAuthorizer and preserves input order.
func (a *Authorizer) BatchCheck(ctx context.Context, reqs []authz.CheckRequest) ([]bool, error) {
	if len(reqs) == 0 {
		return []bool{}, nil
	}
	items := make([]fgaclient.ClientBatchCheckItem, len(reqs))
	for index, req := range reqs {
		if err := validateCheckRequest(req); err != nil {
			return nil, fmt.Errorf("openfga batch check item[%d]: %w", index, err)
		}
		item := fgaclient.ClientBatchCheckItem{
			User:          req.Subject,
			Relation:      req.Action,
			Object:        req.Resource.Type + ":" + req.Resource.ID,
			CorrelationId: strconv.Itoa(index),
		}
		if req.Attributes != nil {
			item.Context = &req.Attributes
		}
		items[index] = item
	}

	response, err := a.client.BatchCheck(ctx).Body(fgaclient.ClientBatchCheckRequest{Checks: items}).Execute()
	if err != nil {
		classified := classifyProviderError(ctx, "batch check", err)
		a.logProviderError(ctx, "batch_check", classified)
		return nil, classified
	}
	if response == nil {
		return nil, fmt.Errorf("openfga batch check: response is nil")
	}
	results := response.GetResult()
	if len(results) != len(reqs) {
		return nil, fmt.Errorf("openfga batch check: result cardinality %d does not match request cardinality %d", len(results), len(reqs))
	}

	allowed := make([]bool, len(reqs))
	for index := range reqs {
		correlationID := strconv.Itoa(index)
		result, ok := results[correlationID]
		if !ok {
			return nil, fmt.Errorf("openfga batch check: missing correlation ID %q", correlationID)
		}
		if checkErr, hasError := result.GetErrorOk(); hasError {
			if code := checkErr.GetInternalError(); code != "" && code != fgasdk.INTERNALERRORCODE_NO_INTERNAL_ERROR {
				cause := fmt.Errorf("openfga batch check item[%d] internal error: %s", index, code)
				return nil, fmt.Errorf("%w: %w", authz.ErrUnavailable, cause)
			}
			if code := checkErr.GetInputError(); code != "" && code != fgasdk.ERRORCODE_NO_ERROR {
				return nil, fmt.Errorf("openfga batch check item[%d] input error: %s", index, code)
			}
			return nil, fmt.Errorf("openfga batch check item[%d] returned an unspecified error", index)
		}
		if !result.HasAllowed() {
			return nil, fmt.Errorf("openfga batch check item[%d] is missing allowed", index)
		}
		allowed[index] = result.GetAllowed()
	}
	return allowed, nil
}

// ListAllowed implements authz.Lister and returns bare resource IDs.
func (a *Authorizer) ListAllowed(ctx context.Context, subject, action, resourceType string) ([]string, error) {
	if err := validateListRequest(subject, action, resourceType); err != nil {
		return nil, err
	}
	response, err := a.client.ListObjects(ctx).Body(fgaclient.ClientListObjectsRequest{
		User:     subject,
		Relation: action,
		Type:     resourceType,
	}).Execute()
	if err != nil {
		classified := classifyProviderError(ctx, "list objects", err)
		a.logProviderError(ctx, "list_objects", classified)
		return nil, classified
	}
	if response == nil {
		return nil, fmt.Errorf("openfga list objects: response is nil")
	}
	objects := response.GetObjects()
	prefix := resourceType + ":"
	ids := make([]string, len(objects))
	for index, object := range objects {
		if !strings.HasPrefix(object, prefix) {
			return nil, fmt.Errorf("openfga list objects: object[%d] %q does not match resource type %q", index, object, resourceType)
		}
		id := strings.TrimPrefix(object, prefix)
		if id == "" {
			return nil, fmt.Errorf("openfga list objects: object[%d] has empty resource ID", index)
		}
		ids[index] = id
	}
	return ids, nil
}

func validateCheckRequest(req authz.CheckRequest) error {
	if strings.TrimSpace(req.Subject) == "" {
		return fmt.Errorf("openfga authz: subject is empty")
	}
	if strings.TrimSpace(req.Action) == "" {
		return fmt.Errorf("openfga authz: action is empty")
	}
	if strings.TrimSpace(req.Resource.Type) == "" {
		return fmt.Errorf("openfga authz: resource type is empty")
	}
	if strings.TrimSpace(req.Resource.ID) == "" {
		return fmt.Errorf("openfga authz: resource ID is empty")
	}
	return nil
}

func validateListRequest(subject, action, resourceType string) error {
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("openfga authz: subject is empty")
	}
	if strings.TrimSpace(action) == "" {
		return fmt.Errorf("openfga authz: action is empty")
	}
	if strings.TrimSpace(resourceType) == "" {
		return fmt.Errorf("openfga authz: resource type is empty")
	}
	return nil
}

func classifyProviderError(ctx context.Context, operation string, err error) error {
	if err == nil {
		return nil
	}
	if contextErr := ctx.Err(); contextErr != nil && !stderrors.Is(err, contextErr) {
		err = stderrors.Join(contextErr, err)
	}
	wrapped := fmt.Errorf("openfga %s: %w", operation, err)
	if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
		return wrapped
	}
	var rateLimit fgasdk.FgaApiRateLimitExceededError
	var internal fgasdk.FgaApiInternalError
	if stderrors.As(err, &rateLimit) || stderrors.As(err, &internal) {
		return fmt.Errorf("%w: %w", authz.ErrUnavailable, wrapped)
	}

	var required fgaclient.FgaRequiredParamError
	var invalid fgaclient.FgaInvalidError
	var validation fgasdk.FgaApiValidationError
	var authentication fgasdk.FgaApiAuthenticationError
	var notFound fgasdk.FgaApiNotFoundError
	var apiError fgasdk.FgaApiError
	var generic fgasdk.GenericOpenAPIError
	var unsupportedType *json.UnsupportedTypeError
	var unsupportedValue *json.UnsupportedValueError
	if stderrors.As(err, &required) || stderrors.As(err, &invalid) ||
		stderrors.As(err, &validation) || stderrors.As(err, &authentication) ||
		stderrors.As(err, &notFound) || stderrors.As(err, &apiError) ||
		stderrors.As(err, &generic) || stderrors.As(err, &unsupportedType) ||
		stderrors.As(err, &unsupportedValue) {
		return wrapped
	}

	// The SDK classifies the remaining Execute errors as network or response
	// body-read failures. Preserve the concrete cause and expose availability.
	return fmt.Errorf("%w: %w", authz.ErrUnavailable, wrapped)
}

func (a *Authorizer) logProviderError(ctx context.Context, operation string, err error) {
	if a.logger == nil {
		return
	}
	reason := "internal"
	switch {
	case stderrors.Is(err, context.Canceled):
		reason = "canceled"
	case stderrors.Is(err, context.DeadlineExceeded):
		reason = "deadline_exceeded"
	case stderrors.Is(err, authz.ErrUnavailable):
		reason = "unavailable"
	}
	a.logger.ErrorContext(ctx, "OpenFGA authorization request failed",
		"operation", operation,
		"reason", reason,
	)
}
