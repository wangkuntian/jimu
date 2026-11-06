package seedmodel_test

import (
	"reflect"
	"testing"

	accessdomain "jimu/internal/capabilities/access/domain"
	tenantdomain "jimu/internal/capabilities/tenant/domain"
	userdomain "jimu/internal/capabilities/user/domain"
	"jimu/internal/kernel/seedmodel"
)

func TestSeedModelMatchesCapabilityDomains(t *testing.T) {
	pairs := []struct {
		name string
		seed any
		real any
	}{
		{"Permission", seedmodel.Permission{}, accessdomain.Permission{}},
		{"Role", seedmodel.Role{}, accessdomain.Role{}},
		{"Tenant", seedmodel.Tenant{}, tenantdomain.Tenant{}},
		{"Plan", seedmodel.Plan{}, tenantdomain.Plan{}},
		{"User", seedmodel.User{}, userdomain.User{}},
	}
	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			assertSamePersistedShape(t, reflect.TypeOf(pair.seed), reflect.TypeOf(pair.real))
		})
	}
}

func assertSamePersistedShape(t *testing.T, seed, real reflect.Type) {
	t.Helper()
	if seed.NumField() != real.NumField() {
		t.Fatalf("%s: field count %d != %d", real.Name(), seed.NumField(), real.NumField())
	}
	for i := 0; i < seed.NumField(); i++ {
		sf, rf := seed.Field(i), real.Field(i)
		if sf.Name != rf.Name {
			t.Fatalf("%s: field %d name %q != %q", real.Name(), i, sf.Name, rf.Name)
		}
		if sf.Type != rf.Type {
			t.Fatalf("%s.%s: type %v != %v", real.Name(), sf.Name, sf.Type, rf.Type)
		}
		if got, want := sf.Tag.Get("gorm"), rf.Tag.Get("gorm"); got != want {
			t.Fatalf("%s.%s: gorm tag %q != %q", real.Name(), sf.Name, got, want)
		}
	}
}

func TestSeedModelTableNamesMatchDomains(t *testing.T) {
	cases := []struct{ seed, real string }{
		{seedmodel.Permission{}.TableName(), accessdomain.Permission{}.TableName()},
		{seedmodel.Role{}.TableName(), accessdomain.Role{}.TableName()},
		{seedmodel.Tenant{}.TableName(), tenantdomain.Tenant{}.TableName()},
		{seedmodel.Plan{}.TableName(), tenantdomain.Plan{}.TableName()},
		{seedmodel.User{}.TableName(), userdomain.User{}.TableName()},
	}
	for _, c := range cases {
		if c.seed != c.real {
			t.Fatalf("table name %q != %q", c.seed, c.real)
		}
	}
}
