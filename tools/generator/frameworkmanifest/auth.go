package frameworkmanifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	"jimu/internal/profiles"
	"jimu/tools/generator/manifest"
)

type authAPITemplateData struct {
	Module            string
	GoVersion         string
	GinVersion        string
	ValidatorVersion  string
	ViperVersion      string
	ZapVersion        string
	LumberjackVersion string
	GormVersion       string
	GormMySQLVersion  string
	RedisVersion      string
	JWTVersion        string
	CryptoVersion     string
}

func authAPIDocument(root, module string) (manifest.Document, error) {
	data, err := loadAuthAPITemplateData(root, module)
	if err != nil {
		return manifest.Document{}, err
	}
	templateData, err := templateDataMap(data)
	if err != nil {
		return manifest.Document{}, err
	}
	templates := authAPITemplates(templateData)
	migrations := []string{
		"internal/migrations/mysql/001_tenants.sql",
		"internal/migrations/mysql/002_users.sql",
		"internal/migrations/mysql/003_roles.sql",
		"internal/migrations/mysql/004_permissions.sql",
		"internal/migrations/mysql/005_role_permissions.sql",
		"internal/migrations/mysql/006_user_roles.sql",
		"internal/migrations/mysql/007_login_histories.sql",
		"internal/migrations/mysql/008_password_histories.sql",
		"internal/migrations/mysql/009_user_last_login.sql",
		"internal/migrations/mysql/010_permissions_updated_at.sql",
	}
	facts := []manifest.Capability{
		{
			Name: "tenant", MigrationOnly: true, Mount: "protected", Owns: []string{"tenants"},
			Migrations: []string{migrations[0]}, Configs: []string{"db"}, RuntimeServices: []string{"mysql"},
		},
		{
			Name: "user", Mount: "protected", Owns: []string{"users"},
			Migrations: []string{migrations[1], migrations[8]}, MigrationSchemaDeps: []string{"tenant"},
			Configs: []string{"db"}, RuntimeServices: []string{"mysql"},
		},
		{
			Name: "access", Mount: "protected",
			Owns:                []string{"roles", "permissions", "role_permissions", "user_roles"},
			Migrations:          []string{migrations[2], migrations[3], migrations[4], migrations[5], migrations[9]},
			MigrationSchemaDeps: []string{"tenant"}, Configs: []string{"db"}, RuntimeServices: []string{"mysql"},
		},
		{
			Name: "auth", Mount: "self-managed", ProtectedHTTPMiddleware: "primary",
			Owns:       []string{"login_histories", "password_histories"},
			Migrations: []string{migrations[6], migrations[7]},
			Configs:    []string{"auth", "db", "redis"}, RuntimeServices: []string{"mysql", "redis"},
		},
	}
	files := []string{"go.sum"}
	for _, action := range templates {
		files = append(files, action.Destination)
	}
	return manifest.Document{
		SchemaVersion: manifest.CurrentSchemaVersion,
		Framework: manifest.Framework{
			Module: "jimu", Version: profiles.Version, Commit: frameworkCommit(root),
		},
		Selection: manifest.Selection{
			Shape: "auth-api", Template: "auth-api", Support: "official",
			Capabilities: []string{"user", "access", "auth"}, Drivers: map[string][]string{},
		},
		Capabilities: facts,
		Copy: []manifest.CopyAction{
			{Source: "go.sum", Destination: "go.sum", Include: []string{}, Exclude: []string{}},
		},
		Templates: templates,
		Merges:    []manifest.MergeAction{}, Rewrites: []manifest.RewriteAction{},
		Assets: []manifest.AssetAction{}, Prune: []manifest.PruneRule{},
		Report: manifest.ReportSpec{
			Name: "auth-api", Capabilities: []string{"user", "access", "auth"}, Routes: 24,
			Migrations: migrations,
			Tables:     []string{"users", "roles", "permissions", "role_permissions", "user_roles", "tenants", "login_histories", "password_histories"},
		},
		GeneratedFiles: files,
	}, nil
}

func loadAuthAPITemplateData(root, module string) (authAPITemplateData, error) {
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return authAPITemplateData{}, fmt.Errorf("read framework go.mod: %w", err)
	}
	file, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return authAPITemplateData{}, fmt.Errorf("parse framework go.mod: %w", err)
	}
	versions := make(map[string]string, len(file.Require))
	for _, requirement := range file.Require {
		versions[requirement.Mod.Path] = requirement.Mod.Version
	}
	data := authAPITemplateData{Module: module}
	if file.Go != nil {
		data.GoVersion = file.Go.Version
	}
	assignments := []struct {
		path string
		dst  *string
	}{
		{"github.com/gin-gonic/gin", &data.GinVersion},
		{"github.com/go-playground/validator/v10", &data.ValidatorVersion},
		{"github.com/spf13/viper", &data.ViperVersion},
		{"go.uber.org/zap", &data.ZapVersion},
		{"gopkg.in/natefinch/lumberjack.v2", &data.LumberjackVersion},
		{"gorm.io/gorm", &data.GormVersion},
		{"gorm.io/driver/mysql", &data.GormMySQLVersion},
		{"github.com/redis/go-redis/v9", &data.RedisVersion},
		{"github.com/golang-jwt/jwt/v5", &data.JWTVersion},
		{"golang.org/x/crypto", &data.CryptoVersion},
	}
	for _, assignment := range assignments {
		version, ok := versions[assignment.path]
		if !ok || version == "" {
			return authAPITemplateData{}, fmt.Errorf("framework go.mod does not require %s", assignment.path)
		}
		*assignment.dst = version
	}
	return data, nil
}

func authAPITemplates(data map[string]any) []manifest.TemplateAction {
	files := []struct {
		source      string
		destination string
	}{
		{"go.mod.tmpl", "go.mod"},
		{"configs/app.yaml.tmpl", "configs/app.yaml"},
		{"README.md.tmpl", "README.md"},
		{"QUICKSTART.md.tmpl", "QUICKSTART.md"},
		{"Makefile.tmpl", "Makefile"},
		{"Dockerfile.tmpl", "Dockerfile"},
		{"docker-compose.yml.tmpl", "docker-compose.yml"},
		{"scripts/init.sh.tmpl", "scripts/init.sh"},
		{"scripts/verify.sh.tmpl", "scripts/verify.sh"},
		{".gitignore.tmpl", ".gitignore"},
		{".golangci.yml.tmpl", ".golangci.yml"},
		{"conf/rbac_model.conf.tmpl", "conf/rbac_model.conf"},
		{"cmd/server/main.go.tmpl", "cmd/server/main.go"},
		{"cmd/cli/main.go.tmpl", "cmd/cli/main.go"},
		{"internal/app/app.go.tmpl", "internal/app/app.go"},
		{"internal/config/config.go.tmpl", "internal/config/config.go"},
		{"internal/domain/models.go.tmpl", "internal/domain/models.go"},
		{"internal/store/db.go.tmpl", "internal/store/db.go"},
		{"internal/store/redis.go.tmpl", "internal/store/redis.go"},
		{"internal/store/migrate.go.tmpl", "internal/store/migrate.go"},
		{"internal/store/seed.go.tmpl", "internal/store/seed.go"},
		{"internal/auth/jwt.go.tmpl", "internal/auth/jwt.go"},
		{"internal/auth/jwt_test.go.tmpl", "internal/auth/jwt_test.go"},
		{"internal/auth/password.go.tmpl", "internal/auth/password.go"},
		{"internal/auth/middleware.go.tmpl", "internal/auth/middleware.go"},
		{"internal/api/v1/routes.go.tmpl", "internal/api/v1/routes.go"},
		{"internal/api/v1/helpers.go.tmpl", "internal/api/v1/helpers.go"},
		{"internal/api/v1/auth.go.tmpl", "internal/api/v1/auth.go"},
		{"internal/api/v1/users.go.tmpl", "internal/api/v1/users.go"},
		{"internal/api/v1/access.go.tmpl", "internal/api/v1/access.go"},
		{"internal/api/v1/routes_test.go.tmpl", "internal/api/v1/routes_test.go"},
		{"internal/migrations/migrate.go.tmpl", "internal/migrations/migrate.go"},
		{"internal/migrations/mysql/001_tenants.sql.tmpl", "internal/migrations/mysql/001_tenants.sql"},
		{"internal/migrations/mysql/002_users.sql.tmpl", "internal/migrations/mysql/002_users.sql"},
		{"internal/migrations/mysql/003_roles.sql.tmpl", "internal/migrations/mysql/003_roles.sql"},
		{"internal/migrations/mysql/004_permissions.sql.tmpl", "internal/migrations/mysql/004_permissions.sql"},
		{"internal/migrations/mysql/005_role_permissions.sql.tmpl", "internal/migrations/mysql/005_role_permissions.sql"},
		{"internal/migrations/mysql/006_user_roles.sql.tmpl", "internal/migrations/mysql/006_user_roles.sql"},
		{"internal/migrations/mysql/007_login_histories.sql.tmpl", "internal/migrations/mysql/007_login_histories.sql"},
		{"internal/migrations/mysql/008_password_histories.sql.tmpl", "internal/migrations/mysql/008_password_histories.sql"},
		{"internal/migrations/mysql/009_user_last_login.sql.tmpl", "internal/migrations/mysql/009_user_last_login.sql"},
		{"internal/migrations/mysql/010_permissions_updated_at.sql.tmpl", "internal/migrations/mysql/010_permissions_updated_at.sql"},
		{"tools/generator/render/templates/basicapi/internal/platform/http/server.go.tmpl", "internal/platform/http/server.go"},
		{"tools/generator/render/templates/basicapi/internal/platform/http/middleware.go.tmpl", "internal/platform/http/middleware.go"},
		{"tools/generator/render/templates/basicapi/internal/platform/health/health.go.tmpl", "internal/platform/health/health.go"},
		{"tools/generator/render/templates/basicapi/internal/platform/logging/logger.go.tmpl", "internal/platform/logging/logger.go"},
		{"tools/generator/render/templates/basicapi/internal/shared/errorx/error.go.tmpl", "internal/shared/errorx/error.go"},
		{"tools/generator/render/templates/basicapi/internal/shared/response/response.go.tmpl", "internal/shared/response/response.go"},
	}
	actions := make([]manifest.TemplateAction, 0, len(files))
	for _, file := range files {
		source := file.source
		if !strings.HasPrefix(source, "tools/") {
			source = filepath.ToSlash(filepath.Join("tools/generator/render/templates/authapi", source))
		}
		actions = append(actions, manifest.TemplateAction{
			Source:      source,
			Destination: file.destination, Kind: "auth-api", Data: data,
		})
	}
	return actions
}
