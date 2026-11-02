package capability

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"jimu/internal/contract"
)

func migrationFS(files ...string) fstest.MapFS {
	out := fstest.MapFS{}
	for _, file := range files {
		out[file] = &fstest.MapFile{Data: []byte("-- migration\n")}
	}
	return out
}

func composeFixture() []contract.Descriptor {
	return []contract.Descriptor{
		{
			Name:                "user",
			Mount:               contract.MountProtected,
			MigrationSchemaDeps: []string{"tenant"},
			Owns:                []string{"users"},
			Migrations:          migrationFS("migrations/mysql/001_users.sql"),
			RuntimeServices:     []contract.RuntimeService{contract.RuntimeServiceMySQL},
			Provides:            []contract.ProvidedPort{{Name: "user.account"}},
		},
		{
			Name:                "access",
			Requires:            []string{"user"},
			Mount:               contract.MountProtected,
			MigrationSchemaDeps: []string{"tenant"},
			Owns:                []string{"roles"},
			Migrations:          migrationFS("migrations/mysql/001_roles.sql"),
			RuntimeServices:     []contract.RuntimeService{contract.RuntimeServiceMySQL},
			Provides:            []contract.ProvidedPort{{Name: "access"}, {Name: "access.provisioning"}},
			Consumes:            []contract.ConsumedPort{{Name: "user.account", Required: true}},
		},
		{
			Name:            "tenant",
			Requires:        []string{"user", "access"},
			Mount:           contract.MountProtected,
			Owns:            []string{"tenants"},
			Migrations:      migrationFS("migrations/mysql/001_tenants.sql"),
			RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceMySQL},
		},
		{
			Name:            "auth",
			Requires:        []string{"user", "access"},
			Mount:           contract.MountSelfManaged,
			RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceMySQL, contract.RuntimeServiceRedis},
			Provides:        []contract.ProvidedPort{{Name: "auth"}},
			Consumes:        []contract.ConsumedPort{{Name: "user.account", Required: true}},

			ProtectedHTTPMiddleware: contract.ProtectedHTTPMiddlewarePrimary,
		},
		{
			Name:            "queue",
			Mount:           contract.MountProtected,
			Drivers:         []string{"redis", "kafka"},
			RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceMySQL},
			Provides:        []contract.ProvidedPort{{Name: "outbox_mq"}},
			DriverSpecs: []contract.DriverSpec{
				{Name: "redis", RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceRedis}},
				{Name: "kafka", RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceKafka}},
			},
		},
	}
}

func TestComposeResolvesClosureDriversCarriersAndServices(t *testing.T) {
	result := Compose(composeFixture(), ComposeRequest{
		Capabilities:   []string{"access"},
		DefaultDrivers: true,
		Support:        contract.SupportExperimental,
	})
	if err := result.Err(); err != nil {
		t.Fatalf("Compose() diagnostics = %v, want valid", err)
	}
	if result.Validity != contract.ValidityValid {
		t.Fatalf("Validity = %q, want valid", result.Validity)
	}
	assertStrings(t, "requested", result.Requested, []string{"access"})
	assertStrings(t, "resolved", descriptorNames(result.Resolved), []string{"user", "access"})
	assertStrings(t, "auto-added", result.AutoAdded, []string{"user"})
	assertStrings(t, "migration carriers", result.MigrationCarriers, []string{"tenant"})
	assertStrings(t, "runtime services", runtimeServiceStrings(result.RuntimeServices), []string{"mysql"})

	withQueue := Compose(composeFixture(), ComposeRequest{
		Capabilities:   []string{"queue"},
		DefaultDrivers: true,
	})
	if err := withQueue.Err(); err != nil {
		t.Fatalf("queue Compose() diagnostics = %v", err)
	}
	if got := withQueue.Drivers["queue"]; len(got) != 1 || got[0] != "redis" {
		t.Fatalf("queue drivers = %v, want [redis]", got)
	}
	assertStrings(t, "queue services", runtimeServiceStrings(withQueue.RuntimeServices), []string{"mysql", "redis"})
}

func TestComposeDiagnosticsByCode(t *testing.T) {
	sharedMigration := map[string][]string{
		"alpha": {"internal/capabilities/alpha/migrations/mysql/001.sql"},
		"beta":  {"internal/capabilities/alpha/migrations/mysql/001.sql"},
	}
	tests := []struct {
		name       string
		caps       []contract.Descriptor
		request    ComposeRequest
		wantCode   DiagnosticCode
		wantPhase  Phase
		wantInText string
	}{
		{
			name:       "unknown capability",
			caps:       composeFixture(),
			request:    ComposeRequest{Capabilities: []string{"ghost"}},
			wantCode:   CodeUnknownCapability,
			wantPhase:  PhaseInput,
			wantInText: `unknown capability "ghost"`,
		},
		{
			name:       "duplicate capability",
			caps:       composeFixture(),
			request:    ComposeRequest{Capabilities: []string{"user", "user"}},
			wantCode:   CodeDuplicateCapability,
			wantPhase:  PhaseInput,
			wantInText: "more than once",
		},
		{
			name: "missing hard dependency",
			caps: composeFixture(),
			request: ComposeRequest{
				Capabilities:             []string{"access"},
				RequireResolvedSelection: true,
			},
			wantCode:   CodeMissingDependency,
			wantPhase:  PhaseClosure,
			wantInText: `requires "user"`,
		},
		{
			name:       "unknown hard dependency",
			caps:       append(composeFixture(), contract.Descriptor{Name: "broken", Requires: []string{"ghost"}}),
			request:    ComposeRequest{Capabilities: []string{"broken"}},
			wantCode:   CodeMissingDependency,
			wantPhase:  PhaseClosure,
			wantInText: `requires unknown capability "ghost"`,
		},
		{
			name: "conflict",
			caps: append(composeFixture(),
				contract.Descriptor{Name: "alpha", Conflicts: []string{"beta"}},
				contract.Descriptor{Name: "beta"},
			),
			request:    ComposeRequest{Capabilities: []string{"alpha", "beta"}},
			wantCode:   CodeConflict,
			wantPhase:  PhaseConflict,
			wantInText: `conflicts with "beta"`,
		},
		{
			name: "duplicate provider",
			caps: append(composeFixture(),
				contract.Descriptor{Name: "alpha", Provides: []contract.ProvidedPort{{Name: "shared"}}},
				contract.Descriptor{Name: "beta", Provides: []contract.ProvidedPort{{Name: "shared"}}},
			),
			request:    ComposeRequest{Capabilities: []string{"alpha", "beta"}},
			wantCode:   CodeDuplicateProvider,
			wantPhase:  PhasePorts,
			wantInText: `port "shared" is provided by both`,
		},
		{
			name: "missing provider",
			caps: append(composeFixture(),
				contract.Descriptor{Name: "alpha", Consumes: []contract.ConsumedPort{{Name: "shared", Required: true}}},
			),
			request:    ComposeRequest{Capabilities: []string{"alpha"}},
			wantCode:   CodeMissingProvider,
			wantPhase:  PhasePorts,
			wantInText: `requires port "shared"`,
		},
		{
			name:       "unknown driver",
			caps:       composeFixture(),
			request:    ComposeRequest{Capabilities: []string{"queue"}, Drivers: map[string][]string{"queue": {"ghost"}}},
			wantCode:   CodeUnknownDriver,
			wantPhase:  PhaseDrivers,
			wantInText: `unknown driver "ghost"`,
		},
		{
			name:       "missing driver selection",
			caps:       composeFixture(),
			request:    ComposeRequest{Capabilities: []string{"queue"}},
			wantCode:   CodeMissingDriver,
			wantPhase:  PhaseDrivers,
			wantInText: "requires a driver selection",
		},
		{
			name: "missing migration schema carrier",
			caps: composeFixture(),
			request: ComposeRequest{
				Capabilities:           []string{"user"},
				CheckMigrationCarriers: true,
			},
			wantCode:   CodeMissingMigrationSchemaDependency,
			wantPhase:  PhaseMigrationSchema,
			wantInText: `migration schema dependency "tenant"`,
		},
		{
			name: "duplicate table owner",
			caps: append(composeFixture(),
				contract.Descriptor{
					Name: "alpha", Owns: []string{"users"},
					Migrations: migrationFS("migrations/mysql/001.sql"),
				},
			),
			request:    ComposeRequest{Capabilities: []string{"alpha", "user"}},
			wantCode:   CodeDuplicateTableOwner,
			wantPhase:  PhaseTables,
			wantInText: `table "users" is owned by both`,
		},
		{
			name: "owned table without migration",
			caps: append(composeFixture(),
				contract.Descriptor{Name: "alpha", Owns: []string{"alpha_rows"}},
			),
			request:    ComposeRequest{Capabilities: []string{"alpha"}},
			wantCode:   CodeMissingMigration,
			wantPhase:  PhaseTables,
			wantInText: "has no migration",
		},
		{
			name: "duplicate migration path",
			caps: append(composeFixture(),
				contract.Descriptor{Name: "alpha"},
				contract.Descriptor{Name: "beta"},
			),
			request: ComposeRequest{
				Capabilities:   []string{"alpha", "beta"},
				MigrationFiles: sharedMigration,
			},
			wantCode:   CodeDuplicateMigration,
			wantPhase:  PhaseTables,
			wantInText: "is included by both",
		},
		{
			name:       "unknown migration carrier",
			caps:       composeFixture(),
			request:    ComposeRequest{Capabilities: []string{"user"}, MigrationCarriers: []string{"ghost"}},
			wantCode:   CodeUnknownCapability,
			wantPhase:  PhaseMigrationSchema,
			wantInText: `migration carrier capability "ghost"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Compose(test.caps, test.request)
			var matched *Diagnostic
			for index := range result.Diagnostics {
				if result.Diagnostics[index].Code == test.wantCode {
					matched = &result.Diagnostics[index]
					break
				}
			}
			if matched == nil {
				t.Fatalf("diagnostics = %#v, want code %q", result.Diagnostics, test.wantCode)
			}
			if matched.Phase != test.wantPhase {
				t.Fatalf("phase = %d, want %d", matched.Phase, test.wantPhase)
			}
			if !strings.Contains(matched.Message, test.wantInText) {
				t.Fatalf("message = %q, want substring %q", matched.Message, test.wantInText)
			}
			if result.Validity != contract.ValidityInvalid {
				t.Fatalf("Validity = %q, want invalid", result.Validity)
			}
		})
	}
}

func TestComposeDiagnosticsAreStable(t *testing.T) {
	request := ComposeRequest{
		Capabilities: []string{"alpha", "beta", "ghost", "queue"},
		Drivers:      map[string][]string{"queue": {"nope"}},
	}
	caps := append(composeFixture(),
		contract.Descriptor{Name: "alpha", Conflicts: []string{"beta"}, Provides: []contract.ProvidedPort{{Name: "shared"}}},
		contract.Descriptor{Name: "beta", Provides: []contract.ProvidedPort{{Name: "shared"}}},
	)
	first := Compose(caps, request)
	second := Compose(caps, request)
	if !reflect.DeepEqual(first.Diagnostics, second.Diagnostics) {
		t.Fatalf("diagnostics differ between runs:\n%#v\n%#v", first.Diagnostics, second.Diagnostics)
	}
	for index := 1; index < len(first.Diagnostics); index++ {
		if first.Diagnostics[index-1].Phase > first.Diagnostics[index].Phase {
			t.Fatalf("diagnostics are not sorted by phase: %#v", first.Diagnostics)
		}
	}
}

// TestComposeStrictModeDoesNotResolveMissingDependencies 钉住检查模式的口径：
// missing_dependency 只报告，不把缺失能力并入 Resolved，否则会掩盖端口/中间件诊断。
func TestComposeStrictModeDoesNotResolveMissingDependencies(t *testing.T) {
	result := Compose(composeFixture(), ComposeRequest{
		Capabilities:             []string{"access"},
		RequireResolvedSelection: true,
	})
	if result.Validity != contract.ValidityInvalid {
		t.Fatal("strict composition with a missing dependency must be invalid")
	}
	if slices.Contains(descriptorNames(result.Resolved), "user") {
		t.Fatalf("strict mode resolved an undeclared dependency: %v", descriptorNames(result.Resolved))
	}
	codes := make(map[DiagnosticCode]bool, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		codes[diagnostic.Code] = true
	}
	if !codes[CodeMissingDependency] || !codes[CodeMissingProvider] {
		t.Fatalf("diagnostics = %#v, want missing_dependency and missing_provider", result.Diagnostics)
	}
}

func TestMigrationCarriersIncludesTransitiveDependencies(t *testing.T) {
	caps := []contract.Descriptor{
		{Name: "a", MigrationSchemaDeps: []string{"b"}},
		{Name: "b", MigrationSchemaDeps: []string{"c"}},
		{Name: "c"},
	}
	got := MigrationCarriers(caps, []string{"a"})
	assertStrings(t, "carriers", got, []string{"b", "c"})
}

func descriptorNames(values []contract.Descriptor) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Name)
	}
	return out
}

func runtimeServiceStrings(values []contract.RuntimeService) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return out
}

func assertStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}
