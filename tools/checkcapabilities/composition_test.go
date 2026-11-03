package main

import (
	"slices"
	"strings"
	"testing"

	"jimu/internal/assembly"
	"jimu/internal/contract"
	"jimu/internal/profiles/registry"
)

// TestCompositionContractAcceptsRealDeclarations 真实 catalog 与全部形态清单必须通过组合契约校验。
func TestCompositionContractAcceptsRealDeclarations(t *testing.T) {
	if err := checkCompositionContract(registry.All()); err != nil {
		t.Fatal(err)
	}
}

// TestCompositionDescriptorsIncludeProfileOnly 非 catalog 能力同样进入组合契约校验面。
func TestCompositionDescriptorsIncludeProfileOnly(t *testing.T) {
	caps, known := compositionDescriptors(registry.All())
	names := make([]string, 0, len(caps))
	for _, d := range caps {
		names = append(names, d.Name)
	}
	for _, want := range []string{"storage", "ws", "encryption", "notification", "grpc", "apidocs"} {
		if !slices.Contains(names, want) {
			t.Fatalf("composition descriptors missing profile-only capability %q: %v", want, names)
		}
		if !slices.Contains(known, want) {
			t.Fatalf("known names missing profile-only capability %q: %v", want, known)
		}
	}
}

// TestCompositionContractChecksEachProfile Required consumer 必须在同一形态内有提供方；
// 只在其他形态出现的提供方不能让本形态通过。
func TestCompositionContractChecksEachProfile(t *testing.T) {
	asms := map[string]assembly.Assembly{
		"consumer-profile": {
			Name: "consumer-profile",
			Capabilities: []assembly.Capability{{
				Descriptor: contract.Descriptor{
					Name:     "consumer",
					Consumes: []contract.ConsumedPort{{Name: "shared", Required: true}},
				},
			}},
		},
		"provider-profile": {
			Name: "provider-profile",
			Capabilities: []assembly.Capability{{
				Descriptor: contract.Descriptor{
					Name:     "provider",
					Provides: []contract.ProvidedPort{{Name: "shared"}},
				},
			}},
		},
	}
	err := checkCompositionContract(asms)
	if err == nil || !strings.Contains(err.Error(), `profile "consumer-profile"`) {
		t.Fatalf("checkCompositionContract() error = %v, want per-profile missing provider", err)
	}
}
