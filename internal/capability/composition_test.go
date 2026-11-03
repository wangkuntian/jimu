package capability

import (
	"strings"
	"testing"

	"jimu/internal/contract"
)

func TestValidateDeclarationsComposition(t *testing.T) {
	tests := []struct {
		name    string
		caps    []contract.Descriptor
		known   []string
		wantErr string
	}{
		{
			name: "valid declarations",
			caps: []contract.Descriptor{
				{
					Name:     "queue",
					Drivers:  []string{"redis"},
					Provides: []contract.ProvidedPort{{Name: "outbox_mq"}},
					DriverSpecs: []contract.DriverSpec{
						{Name: "redis", RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceRedis}},
					},
				},
				{
					Name:      "outbox",
					Consumes:  []contract.ConsumedPort{{Name: "outbox_mq"}},
					Conflicts: []string{"legacy_outbox"},
				},
				{Name: "legacy_outbox"},
			},
			known: []string{"queue", "outbox", "legacy_outbox"},
		},
		{
			name:    "empty conflict",
			caps:    []contract.Descriptor{{Name: "a", Conflicts: []string{""}}},
			known:   []string{"a"},
			wantErr: "empty conflict",
		},
		{
			name:    "self conflict",
			caps:    []contract.Descriptor{{Name: "a", Conflicts: []string{"a"}}},
			known:   []string{"a"},
			wantErr: "conflicts with itself",
		},
		{
			name:    "duplicate conflict",
			caps:    []contract.Descriptor{{Name: "a", Conflicts: []string{"b", "b"}}, {Name: "b"}},
			known:   []string{"a", "b"},
			wantErr: "duplicates conflict",
		},
		{
			name:    "unknown conflict",
			caps:    []contract.Descriptor{{Name: "a", Conflicts: []string{"missing"}}},
			known:   []string{"a"},
			wantErr: "conflicts with unknown capability",
		},
		{
			name: "duplicate provider across capabilities",
			caps: []contract.Descriptor{
				{Name: "a", Provides: []contract.ProvidedPort{{Name: "shared"}}},
				{Name: "b", Provides: []contract.ProvidedPort{{Name: "shared"}}},
			},
			known:   []string{"a", "b"},
			wantErr: "is provided by both",
		},
		{
			name:    "duplicate provided port within capability",
			caps:    []contract.Descriptor{{Name: "a", Provides: []contract.ProvidedPort{{Name: "p"}, {Name: "p"}}}},
			known:   []string{"a"},
			wantErr: "duplicates provided port",
		},
		{
			name:    "empty provided port",
			caps:    []contract.Descriptor{{Name: "a", Provides: []contract.ProvidedPort{{}}}},
			known:   []string{"a"},
			wantErr: "empty provided port",
		},
		{
			name:    "RequiredFlag consumer without provider",
			caps:    []contract.Descriptor{{Name: "a", Consumes: []contract.ConsumedPort{{Name: "p", Required: true}}}},
			known:   []string{"a"},
			wantErr: "no capability provides it",
		},
		{
			name:  "optional consumer without provider",
			caps:  []contract.Descriptor{{Name: "a", Consumes: []contract.ConsumedPort{{Name: "p"}}}},
			known: []string{"a"},
		},
		{
			name:    "empty consumed port",
			caps:    []contract.Descriptor{{Name: "a", Consumes: []contract.ConsumedPort{{}}}},
			known:   []string{"a"},
			wantErr: "empty consumed port",
		},
		{
			name:    "duplicate consumed port",
			caps:    []contract.Descriptor{{Name: "a", Consumes: []contract.ConsumedPort{{Name: "p"}, {Name: "p"}}}},
			known:   []string{"a"},
			wantErr: "duplicates consumed port",
		},
		{
			name:    "empty driver spec name",
			caps:    []contract.Descriptor{{Name: "a", DriverSpecs: []contract.DriverSpec{{}}}},
			known:   []string{"a"},
			wantErr: "empty driver spec name",
		},
		{
			name: "duplicate driver spec",
			caps: []contract.Descriptor{{
				Name:        "a",
				Drivers:     []string{"local"},
				DriverSpecs: []contract.DriverSpec{{Name: "local"}, {Name: "local"}},
			}},
			known:   []string{"a"},
			wantErr: "duplicates driver spec",
		},
		{
			name: "driver spec outside drivers",
			caps: []contract.Descriptor{{
				Name:        "a",
				Drivers:     []string{"local"},
				DriverSpecs: []contract.DriverSpec{{Name: "s3"}},
			}},
			known:   []string{"a"},
			wantErr: "outside Drivers",
		},
		{
			name: "unknown driver runtime service",
			caps: []contract.Descriptor{{
				Name:        "a",
				Drivers:     []string{"local"},
				DriverSpecs: []contract.DriverSpec{{Name: "local", RuntimeServices: []contract.RuntimeService{"nats"}}},
			}},
			known:   []string{"a"},
			wantErr: "unknown runtime service",
		},
		{
			name: "duplicate driver runtime service",
			caps: []contract.Descriptor{{
				Name:    "a",
				Drivers: []string{"local"},
				DriverSpecs: []contract.DriverSpec{{
					Name:            "local",
					RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceRedis, contract.RuntimeServiceRedis},
				}},
			}},
			known:   []string{"a"},
			wantErr: "duplicates runtime service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeclarations(tt.caps, tt.known)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateDeclarations() error = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateDeclarations() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestResolveDeepCopiesCompositionFields(t *testing.T) {
	caps := []contract.Descriptor{{
		Name:      "a",
		Conflicts: []string{"b"},
		Provides:  []contract.ProvidedPort{{Name: "p"}},
		Consumes:  []contract.ConsumedPort{{Name: "q", Required: true}},
		Drivers:   []string{"redis"},
		DriverSpecs: []contract.DriverSpec{
			{Name: "redis", RuntimeServices: []contract.RuntimeService{contract.RuntimeServiceRedis}},
		},
	}}

	got, err := Resolve(caps, nil)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	got[0].Conflicts[0] = "mutated"
	got[0].Provides[0].Name = "mutated"
	got[0].Consumes[0].Name = "mutated"
	got[0].DriverSpecs[0].Name = "mutated"
	got[0].DriverSpecs[0].RuntimeServices[0] = contract.RuntimeService("mutated")

	again, err := Resolve(caps, nil)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if again[0].Conflicts[0] != "b" {
		t.Fatalf("Conflicts backing array leaked: %q", again[0].Conflicts[0])
	}
	if again[0].Provides[0].Name != "p" {
		t.Fatalf("Provides backing array leaked: %q", again[0].Provides[0].Name)
	}
	if again[0].Consumes[0].Name != "q" {
		t.Fatalf("Consumes backing array leaked: %q", again[0].Consumes[0].Name)
	}
	if again[0].DriverSpecs[0].Name != "redis" {
		t.Fatalf("DriverSpecs backing array leaked: %q", again[0].DriverSpecs[0].Name)
	}
	if again[0].DriverSpecs[0].RuntimeServices[0] != contract.RuntimeServiceRedis {
		t.Fatalf("DriverSpecs.RuntimeServices backing array leaked: %q", again[0].DriverSpecs[0].RuntimeServices[0])
	}
}
