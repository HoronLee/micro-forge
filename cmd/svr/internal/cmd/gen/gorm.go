package gen

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/horonlee/servora/cmd/svr/internal/discovery"
	"github.com/horonlee/servora/cmd/svr/internal/generator"
	"github.com/horonlee/servora/cmd/svr/internal/ux"
)

// Error type constants for structured failure reporting.
const (
	errServiceNotFound  = "service-not-found"
	errConfigInvalid    = "config-invalid"
	errDBConnectFailed  = "db-connect-failed"
	errGenerationFailed = "generation-failed"
)

// serviceFailure records a single service generation failure.
type serviceFailure struct {
	service   string
	errorType string
	message   string
}

// NewGormCmd creates the "gorm" subcommand for GORM GEN code generation.
func NewGormCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "gorm [service-name...]",
		Short: "Generate GORM GEN code for services",
		Long:  "Generate GORM GEN DAO and PO code from database schema. Supports multiple services and interactive selection.",
		RunE: func(cmd *cobra.Command, args []string) error {
			services := discovery.NormalizeServiceNames(args)

			// If no args, enter interactive mode.
			if len(services) == 0 {
				selected, err := interactiveSelect()
				if err != nil {
					return err
				}
				services = selected
			}

			return runBatch(services, dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show planned output paths without generating code")

	return cmd
}

// interactiveSelect presents a multi-select UI for choosing services.
func interactiveSelect() ([]string, error) {
	available, err := discovery.ListAvailableServices()
	if err != nil {
		ux.PrintError("Discovery", fmt.Sprintf("failed to list services: %v", err))
		return nil, err
	}

	if len(available) == 0 {
		ux.PrintInfo("No services found in app/*/service")
		return nil, nil
	}

	options := make([]huh.Option[string], 0, len(available))
	for _, svc := range available {
		options = append(options, huh.NewOption(svc, svc))
	}

	var selected []string
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select services for GORM GEN").
				Options(options...).
				Value(&selected),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Generate GORM code for selected services?").
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed),
		),
	)

	err = form.Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			ux.PrintInfo("Cancelled by user")
			return nil, nil
		}
		return nil, err
	}

	if !confirmed {
		ux.PrintInfo("Cancelled by user")
		return nil, nil
	}

	if len(selected) == 0 {
		ux.PrintInfo("No services selected")
		return nil, nil
	}

	return selected, nil
}

// runBatch executes generation for multiple services, collecting failures.
func runBatch(services []string, dryRun bool) error {
	total := len(services)
	if total == 0 {
		return nil
	}

	var (
		successCount int
		failures     []serviceFailure
	)

	for i, svc := range services {
		ux.PrintProgress(i+1, total, fmt.Sprintf("generating %s", svc))

		if err := generateForService(svc, dryRun); err != nil {
			failures = append(failures, *err)
		} else {
			successCount++
		}
	}

	failedCount := len(failures)
	ux.PrintSummary(successCount, failedCount)

	if failedCount > 0 {
		for _, f := range failures {
			ux.PrintFailureDetail(f.service, f.errorType, f.message)
		}
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}

	return nil
}

// generateForService orchestrates generation for a single service.
// Returns a *serviceFailure on error, nil on success.
func generateForService(serviceName string, dryRun bool) *serviceFailure {
	// 1. Validate service exists.
	if err := discovery.ValidateServiceExists(serviceName); err != nil {
		available, _ := discovery.ListAvailableServices()
		msg := err.Error()
		if len(available) > 0 {
			msg += fmt.Sprintf("\n  available services: %s", strings.Join(available, ", "))
		}
		ux.PrintError(serviceName, msg)
		return &serviceFailure{
			service:   serviceName,
			errorType: errServiceNotFound,
			message:   err.Error(),
		}
	}

	// 2. Validate config exists.
	if err := discovery.ValidateConfigExists(serviceName); err != nil {
		ux.PrintError(serviceName, err.Error())
		return &serviceFailure{
			service:   serviceName,
			errorType: errConfigInvalid,
			message:   err.Error(),
		}
	}

	// 3. Load service config.
	svcCfg, err := discovery.LoadServiceConfig(serviceName)
	if err != nil {
		ux.PrintError(serviceName, fmt.Sprintf("failed to load config: %v", err))
		return &serviceFailure{
			service:   serviceName,
			errorType: errConfigInvalid,
			message:   err.Error(),
		}
	}

	// 4. Validate database config.
	if err := discovery.ValidateDatabaseConfig(svcCfg.Bootstrap); err != nil {
		msg := fmt.Sprintf("no database config found in app/%s/service/configs/config.yaml", serviceName)
		hint := "\n  expected config.yaml to contain:\n    data:\n      database:\n        driver: mysql\n        source: <dsn>"
		ux.PrintError(serviceName, msg+hint)
		return &serviceFailure{
			service:   serviceName,
			errorType: errConfigInvalid,
			message:   msg,
		}
	}

	// 5. Run generator.
	g := &generator.GormGenerator{
		ServiceName: serviceName,
		ServicePath: svcCfg.Path,
		DatabaseCfg: svcCfg.Bootstrap.GetData().GetDatabase(),
		DryRun:      dryRun,
	}

	if err := g.Generate(); err != nil {
		errMsg := err.Error()
		errType := errGenerationFailed
		if strings.Contains(errMsg, "connect db failed") || strings.Contains(errMsg, "unsupported db driver") {
			errType = errDBConnectFailed
			hint := "\n  checklist:\n  - Is the database server running?\n  - Is the DSN in config.yaml correct?\n  - Is the driver supported? (mysql, postgres, sqlite)"
			ux.PrintError(serviceName, errMsg+hint)
		} else {
			ux.PrintError(serviceName, errMsg)
		}
		return &serviceFailure{
			service:   serviceName,
			errorType: errType,
			message:   errMsg,
		}
	}

	return nil
}
