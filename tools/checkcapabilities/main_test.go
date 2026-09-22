package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOwnershipViolations(t *testing.T) {
	tests := []struct {
		name     string
		declared map[string][]string
		created  map[string][]string
		wantErr  bool
	}{
		{
			name:     "declared but not created",
			declared: map[string][]string{"user": {"users"}},
			created:  map[string][]string{},
			wantErr:  true,
		},
		{
			name:     "created but not declared",
			declared: map[string][]string{},
			created:  map[string][]string{"user": {"users"}},
			wantErr:  true,
		},
		{
			name:     "created by another capability",
			declared: map[string][]string{"user": {"users"}},
			created:  map[string][]string{"access": {"users"}},
			wantErr:  true,
		},
		{
			name:     "owned by two capabilities",
			declared: map[string][]string{"user": {"users"}, "access": {"users"}},
			created:  map[string][]string{"user": {"users"}, "access": {"users"}},
			wantErr:  true,
		},
		{
			name:     "consistent",
			declared: map[string][]string{"user": {"users"}},
			created:  map[string][]string{"user": {"users"}},
			wantErr:  false,
		},
		{
			name:     "no-op capability",
			declared: map[string][]string{},
			created:  map[string][]string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkOwnership(tt.declared, tt.created)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
