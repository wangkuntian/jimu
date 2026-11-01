package preflight

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"jimu/tools/generator/manifest"
)

func validDocument() manifest.Document {
	return manifest.Document{
		SchemaVersion: manifest.CurrentSchemaVersion,
		Selection: manifest.Selection{
			Shape:        "app",
			Capabilities: []string{"user", "access", "auth"},
		},
		Capabilities: []manifest.Capability{
			{
				Name:                    "tenant",
				MigrationOnly:           true,
				Owns:                    []string{"tenants"},
				Migrations:              []string{"internal/capabilities/tenant/migrations/mysql/005_tenants.sql"},
				RuntimeServices:         []string{"mysql"},
				ProtectedHTTPMiddleware: "",
				Mount:                   "protected",
			},
			{
				Name:                    "user",
				Owns:                    []string{"users"},
				Migrations:              []string{"internal/capabilities/user/migrations/mysql/001_users.sql"},
				RuntimeServices:         []string{"mysql", "redis"},
				ProtectedHTTPMiddleware: "",
				Mount:                   "protected",
			},
			{
				Name:                    "access",
				Owns:                    []string{"roles"},
				Migrations:              []string{"internal/capabilities/access/migrations/mysql/001_roles.sql"},
				MigrationSchemaDeps:     []string{"tenant"},
				RuntimeServices:         []string{"mysql"},
				ProtectedHTTPMiddleware: "",
				Mount:                   "protected",
			},
			{
				Name:                    "auth",
				RuntimeServices:         []string{"mysql", "redis"},
				ProtectedHTTPMiddleware: "primary",
				Mount:                   "self-managed",
			},
		},
		Merges: []manifest.MergeAction{
			{Destination: "configs/app.yaml", Sections: []string{"db", "redis", "log"}},
			{Destination: "configs/app.prod.yaml", Sections: []string{"db", "redis", "log"}},
		},
	}
}

func TestValidateAcceptsConsistentSelection(t *testing.T) {
	require.NoError(t, Validate(validDocument()))
}

func TestValidateAcceptsFallbackProviderWhenPrimaryIsAbsent(t *testing.T) {
	doc := validDocument()
	doc.Selection.Capabilities = []string{"user", "apikey"}
	doc.Capabilities[0].MigrationOnly = true
	doc.Capabilities = append(doc.Capabilities, manifest.Capability{
		Name:                    "apikey",
		Owns:                    []string{"api_keys"},
		Migrations:              []string{"internal/capabilities/apikey/migrations/mysql/001_api_keys.sql"},
		RuntimeServices:         []string{"mysql"},
		ProtectedHTTPMiddleware: "fallback",
		Mount:                   "protected",
	})

	require.NoError(t, Validate(doc))
}

func TestValidateRejectsMissingProtectedMiddlewareProvider(t *testing.T) {
	doc := validDocument()
	doc.Selection.Capabilities = []string{"user", "access"}
	for i := range doc.Capabilities {
		if doc.Capabilities[i].Name == "auth" {
			doc.Capabilities = append(doc.Capabilities[:i], doc.Capabilities[i+1:]...)
			break
		}
	}

	err := Validate(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "protected middleware")
}

func TestValidateRejectsMultiplePrimaryProtectedMiddlewareProviders(t *testing.T) {
	doc := validDocument()
	doc.Selection.Capabilities = append(doc.Selection.Capabilities, "second-auth")
	doc.Capabilities = append(doc.Capabilities, manifest.Capability{
		Name:                    "second-auth",
		ProtectedHTTPMiddleware: "primary",
		Mount:                   "self-managed",
	})

	err := Validate(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple primary")
}

func TestValidateRejectsMissingMigrationSchemaDependency(t *testing.T) {
	doc := validDocument()
	doc.Capabilities = doc.Capabilities[1:] // drop tenant carrier

	err := Validate(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "migration schema dependency")
	require.Contains(t, err.Error(), "tenant")
}

func TestValidateRejectsDuplicateTableOwnership(t *testing.T) {
	doc := validDocument()
	doc.Capabilities[1].Owns = append(doc.Capabilities[1].Owns, "tenants")

	err := Validate(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), `table "tenants"`)
}

func TestValidateRejectsOwnedTableWithoutMigration(t *testing.T) {
	doc := validDocument()
	doc.Capabilities[1].Migrations = nil

	err := Validate(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "owns table")
}

func TestValidateRejectsRuntimeServiceWithoutConfiguration(t *testing.T) {
	doc := validDocument()
	doc.Merges[0].Sections = []string{"redis", "log"}

	err := Validate(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mysql")
	require.Contains(t, err.Error(), "db")
}

func TestValidateRejectsUnknownRuntimeService(t *testing.T) {
	doc := validDocument()
	doc.Capabilities[3].RuntimeServices = []string{"postgres"}

	err := Validate(doc)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "unknown runtime service"))
}
