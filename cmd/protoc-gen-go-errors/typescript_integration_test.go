package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedTypeScriptCompilesAndNarrows(t *testing.T) {
	if testing.Short() {
		t.Skip("requires the proto-utils TypeScript toolchain")
	}

	generated, err := generateWithParameter(
		t,
		typeScriptFixture(),
		"paths=source_relative,target=ts",
	)
	if err != nil {
		t.Fatalf("generate target=ts: %v", err)
	}
	const filename = "example/service/v1/user.errors.ts"
	sidecar, ok := generated[filename]
	if !ok {
		t.Fatalf("generated files = %v, want %s", generated, filename)
	}

	fixtureDir := t.TempDir()
	writeTypeScriptFixture(t, filepath.Join(fixtureDir, "package.json"), `{"type":"module"}`)
	writeTypeScriptFixture(t, filepath.Join(fixtureDir, "index.ts"), typeScriptIndexFixture)
	writeTypeScriptFixture(t, filepath.Join(fixtureDir, "user.errors.ts"), sidecar)
	writeTypeScriptFixture(t, filepath.Join(fixtureDir, "contract.ts"), typeScriptContractFixture)
	writeTypeScriptFixture(t, filepath.Join(fixtureDir, "tsconfig.bundler.json"), bundlerConfigFixture)
	writeTypeScriptFixture(t, filepath.Join(fixtureDir, "tsconfig.nodenext.json"), nodeNextConfigFixture)
	writeTypeScriptFixture(t, filepath.Join(fixtureDir, "tsconfig.declarations.json"), declarationConfigFixture)

	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	protoUtilsDir := filepath.Join(repositoryRoot, "web", "packages", "proto-utils")
	for _, config := range []string{"tsconfig.bundler.json", "tsconfig.nodenext.json"} {
		runTypeScriptFixtureCommand(
			t,
			protoUtilsDir,
			"pnpm",
			"--dir",
			protoUtilsDir,
			"exec",
			"tsc",
			"--project",
			filepath.Join(fixtureDir, config),
		)
	}
	runTypeScriptFixtureCommand(
		t,
		protoUtilsDir,
		"pnpm",
		"--dir",
		protoUtilsDir,
		"exec",
		"tsc",
		"--project",
		filepath.Join(fixtureDir, "tsconfig.declarations.json"),
	)
	runTypeScriptFixtureCommand(
		t,
		fixtureDir,
		"node",
		filepath.Join(fixtureDir, "out", "contract.js"),
	)

	runtimeSource := readTypeScriptFixture(t, filepath.Join(fixtureDir, "out", "user.errors.js"))
	if strings.Contains(runtimeSource, "./index.js") || strings.Contains(runtimeSource, "import type") {
		t.Fatalf("runtime JavaScript retained type-only import:\n%s", runtimeSource)
	}
	declarationSource := readTypeScriptFixture(t, filepath.Join(fixtureDir, "out", "user.errors.d.ts"))
	if !strings.Contains(declarationSource, "from './index.js'") {
		t.Fatalf("declaration does not retain NodeNext-resolvable .js type import:\n%s", declarationSource)
	}
}

func writeTypeScriptFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readTypeScriptFixture(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func runTypeScriptFixtureCommand(t *testing.T, directory, name string, args ...string) {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = directory
	command.Env = os.Environ()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, output)
	}
}

const typeScriptIndexFixture = `export type UserErrorReason =
  | 'USER_ERROR_REASON_UNSPECIFIED'
  | 'USER_ERROR_REASON_INVALID_NAME'
  | 'USER_ERROR_REASON_INTERNAL';

export type PaymentErrorReason =
  | 'PAYMENT_ERROR_REASON_UNSPECIFIED'
  | 'PAYMENT_ERROR_REASON_DECLINED';
`

const typeScriptContractFixture = `import {
  isPaymentErrorReason,
  isUserErrorReason,
  PaymentErrorReason,
  UserErrorReason,
} from './user.errors.js';

const compileTimeReason: UserErrorReason =
  UserErrorReason.USER_ERROR_REASON_INVALID_NAME;
const paymentReason: PaymentErrorReason =
  PaymentErrorReason.PAYMENT_ERROR_REASON_UNSPECIFIED;

function narrowUserReason(value: unknown): UserErrorReason | undefined {
  if (isUserErrorReason(value)) {
    const narrowed: UserErrorReason = value;
    return narrowed;
  }
  return undefined;
}

if (narrowUserReason(compileTimeReason) !== compileTimeReason) {
  throw new Error('valid user reason did not narrow');
}
if (!isPaymentErrorReason(paymentReason)) {
  throw new Error('value-level error enum omitted an unannotated member');
}

const invalidValues: unknown[] = [
  'USER_ERROR_REASON_UNKNOWN',
  'user_error_reason_invalid_name',
  'toString',
  null,
  1,
  {},
  Object.create({ USER_ERROR_REASON_INVALID_NAME: true }),
];
for (const value of invalidValues) {
  if (isUserErrorReason(value)) {
    throw new Error('invalid user reason accepted: ' + String(value));
  }
}
`

const bundlerConfigFixture = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "verbatimModuleSyntax": true,
    "strict": true,
    "noEmit": true
  },
  "include": ["*.ts"]
}`

const nodeNextConfigFixture = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "verbatimModuleSyntax": true,
    "strict": true,
    "declaration": true,
    "outDir": "./out",
    "rootDir": "."
  },
  "include": ["*.ts"]
}`

const declarationConfigFixture = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "verbatimModuleSyntax": true,
    "strict": true,
    "noEmit": true,
    "skipLibCheck": false
  },
  "include": ["out/**/*.d.ts"]
}`
