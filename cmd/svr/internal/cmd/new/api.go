package new

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/Servora-Kit/servora/cmd/svr/internal/discovery"
)

var nameRegex = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*(\.[a-z][a-z0-9]*(_[a-z0-9]+)*)*$`)

// defaultTemplateDir is the project-local template directory resolved relative to cwd.
// svr is designed to run from the project root, so this path is always valid.
const defaultTemplateDir = "api/protos/template/service/v1"

// NewApiCmd creates the "new api" subcommand.
func NewApiCmd() *cobra.Command {
	var templateDir string

	cmd := &cobra.Command{
		Use:   "api <name> <server_name>",
		Short: "Scaffold a new proto API in a service directory",
		Long: `Scaffold a new gRPC service proto skeleton under app/<server_name>/service/api/protos/.

Name must be lowercase snake_case, optionally dot-separated for nesting:
  svr new api billing servora
  svr new api say_hello sayhello
  svr new api billing.invoice servora

Server name must correspond to an existing service under app/<server_name>/service.

Must be run from the project root directory.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			serverName := args[1]

			if err := validateName(name); err != nil {
				return err
			}

			// Validate server exists using discovery
			services, err := discovery.ListAvailableServices()
			if err != nil {
				return fmt.Errorf("failed to discover services: %w", err)
			}

			found := false
			for _, svc := range services {
				if svc == serverName {
					found = true
					break
				}
			}

			if !found {
				fmt.Fprintf(os.Stderr, "error: service %q not found\n", serverName)
				fmt.Fprintf(os.Stderr, "\nAvailable services:\n")
				for _, svc := range services {
					fmt.Fprintf(os.Stderr, "  - %s\n", svc)
				}
				return fmt.Errorf("service %q does not exist", serverName)
			}

			// Build output directory: app/<server_name>/service/api/protos/<name>/service/v1/
			outputRoot := filepath.Join("app", serverName, "service", "api", "protos")
			return runNewApi(name, outputRoot, templateDir)
		},
	}

	cmd.Flags().StringVar(&templateDir, "template", "", "Custom template directory (must contain template.proto and template_doc.proto)")

	return cmd
}

// validateName checks the name conforms to proto package naming rules.
func validateName(name string) error {
	if !nameRegex.MatchString(name) {
		return fmt.Errorf(
			"invalid name %q: must be lowercase snake_case, optionally dot-separated (e.g. test, say_hello, billing.invoice)",
			name,
		)
	}
	return nil
}

// runNewApi orchestrates the scaffolding.
func runNewApi(name, outputRoot, templateDir string) error {
	// Compute target directory and file base name.
	segments := strings.Split(name, ".")
	dirPath := filepath.Join(append([]string{outputRoot}, append(segments, "service", "v1")...)...)
	fileBase := strings.Join(segments, "_") // test.test1 → test_test1

	// Conflict check.
	if _, err := os.Stat(dirPath); err == nil {
		return fmt.Errorf("directory already exists: %s", dirPath)
	}

	// Load templates.
	mainTmpl, docTmpl, err := loadTemplates(templateDir)
	if err != nil {
		return err
	}

	// Apply naming substitutions.
	mainContent := applySubstitutions(string(mainTmpl), name)
	docContent := applySubstitutions(string(docTmpl), name)

	// Create target directory.
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Write proto files.
	mainFile := filepath.Join(dirPath, fileBase+".proto")
	docFile := filepath.Join(dirPath, fileBase+"_doc.proto")

	if err := os.WriteFile(mainFile, []byte(mainContent), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", mainFile, err)
	}
	if err := os.WriteFile(docFile, []byte(docContent), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", docFile, err)
	}

	fmt.Printf("✓ Created:\n  %s\n  %s\n", mainFile, docFile)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Run 'just gen' to generate Go code\n")
	fmt.Printf("  2. If you need OpenAPI/TypeScript generation, check service-level config:\n")
	fmt.Printf("     - api/buf.openapi.gen.yaml\n")
	fmt.Printf("     - api/buf.typescript.gen.yaml\n")
	return nil
}

// loadTemplates returns the contents of template.proto and template_doc.proto.
// Priority: --template flag → default project template (api/protos/template/service/v1/).
// svr is designed to run from the project root, so the default path is always resolvable.
func loadTemplates(flagDir string) (main []byte, doc []byte, err error) {
	dir := defaultTemplateDir
	if flagDir != "" {
		dir = flagDir
	}

	mainPath := filepath.Join(dir, "template.proto")
	docPath := filepath.Join(dir, "template_doc.proto")

	main, err = os.ReadFile(mainPath)
	if err != nil {
		return nil, nil, fmt.Errorf("template not found: %s\n  hint: run svr from the project root, or use --template to specify a directory", mainPath)
	}
	doc, err = os.ReadFile(docPath)
	if err != nil {
		return nil, nil, fmt.Errorf("template not found: %s\n  hint: run svr from the project root, or use --template to specify a directory", docPath)
	}
	return main, doc, nil
}

// applySubstitutions replaces template placeholder words with the target name variants.
//
// Proto naming rules:
//   - package name uses dot-separated form matching the directory structure
//     e.g. "billing.invoice.service.v1" for dir billing/invoice/service/v1
//   - java_package also uses dot-separated name segment
//   - file names and Go/Java class names use snake/pascal forms
//
// Replacement order: most-specific patterns first to prevent partial matches.
func applySubstitutions(content, name string) string {
	snake := toSnake(name)   // billing_invoice  (file names, option identifiers)
	pascal := toPascal(name) // BillingInvoice   (service/message/Java class names)
	upper := toUpper(name)   // BILLING_INVOICE  (screaming snake, if any)

	// 1. proto package line: "package template." → "package <name>."
	//    Preserves dot-separated form required by buf/protoc directory matching.
	content = strings.ReplaceAll(content, "package template.", "package "+name+".")

	// 2. java_package option: ".api.template.v1" → ".api.<name>.v1"
	//    Uses dot-separated name so Java package mirrors proto package structure.
	content = strings.ReplaceAll(content, ".api.template.", ".api."+name+".")

	// 3. PascalCase identifiers (service names, message names, Java outer class).
	content = strings.ReplaceAll(content, "Template", pascal)

	// 4. SCREAMING_SNAKE identifiers (if any).
	content = strings.ReplaceAll(content, "TEMPLATE", upper)

	// 5. Remaining lowercase occurrences (title strings, descriptions, file refs).
	content = strings.ReplaceAll(content, "template", snake)

	return content
}

// toSnake returns the snake_case file-name form of the name.
// Dots are replaced with underscores: "test.test1" → "test_test1".
func toSnake(name string) string {
	return strings.ReplaceAll(name, ".", "_")
}

// toPascal converts a dot/snake name to PascalCase.
// "say_hello" → "SayHello", "test.test1" → "TestTest1".
func toPascal(name string) string {
	var b strings.Builder
	for _, seg := range strings.Split(name, ".") {
		for _, word := range strings.Split(seg, "_") {
			if len(word) == 0 {
				continue
			}
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			b.WriteString(string(runes))
		}
	}
	return b.String()
}

// toUpper converts a dot/snake name to SCREAMING_SNAKE_CASE.
// "say_hello" → "SAY_HELLO", "test.test1" → "TEST_TEST1".
func toUpper(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, ".", "_"))
}
