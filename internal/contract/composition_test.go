package contract

import "testing"

func TestValidityRecognized(t *testing.T) {
	tests := []struct {
		value Validity
		want  bool
	}{
		{ValidityValid, true},
		{ValidityInvalid, true},
		{Validity(""), false},
		{Validity("Valid"), false},
		{Validity("unknown"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.value), func(t *testing.T) {
			if got := tt.value.Valid(); got != tt.want {
				t.Fatalf("%q.Valid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestSupportLevelRecognized(t *testing.T) {
	tests := []struct {
		value SupportLevel
		want  bool
	}{
		{SupportCertified, true},
		{SupportValidated, true},
		{SupportExperimental, true},
		{SupportLevel(""), false},
		{SupportLevel("official"), false},
		{SupportLevel("Certified"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.value), func(t *testing.T) {
			if got := tt.value.Valid(); got != tt.want {
				t.Fatalf("%q.Valid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
