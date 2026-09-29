package main

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"jimu/internal/contract"

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

// TestCreatedTables CREATE TABLE 提取：容忍任意空白、IF NOT EXISTS、TEMPORARY，
// 多文件去重排序，表名大小写归一。
func TestCreatedTables(t *testing.T) {
	tests := []struct {
		name string
		fsys fs.FS
		want []string
	}{
		{
			name: "plain create table",
			fsys: fstest.MapFS{
				"migrations/mysql/001_users.sql": {Data: []byte("CREATE TABLE users (id BIGINT PRIMARY KEY);\n")},
			},
			want: []string{"users"},
		},
		{
			name: "create table if not exists",
			fsys: fstest.MapFS{
				"migrations/mysql/001_users.sql": {Data: []byte("CREATE TABLE IF NOT EXISTS users (id BIGINT);\n")},
			},
			want: []string{"users"},
		},
		{
			name: "create temporary table",
			fsys: fstest.MapFS{
				"migrations/mysql/001_tmp.sql": {Data: []byte("CREATE TEMPORARY TABLE tmp_import (id BIGINT);\n")},
			},
			want: []string{"tmp_import"},
		},
		{
			name: "multiple files deduped and sorted",
			fsys: fstest.MapFS{
				"migrations/mysql/001_a.sql": {Data: []byte("CREATE TABLE zebra (id BIGINT);\nCREATE   TABLE   alpha (id BIGINT);\n")},
				"migrations/mysql/002_b.sql": {Data: []byte("create table IF NOT EXISTS Alpha (id BIGINT);\n")},
			},
			want: []string{"alpha", "zebra"},
		},
		{
			name: "non-sql files ignored",
			fsys: fstest.MapFS{
				"migrations/mysql/README.md": {Data: []byte("CREATE TABLE ignored (id BIGINT);\n")},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createdTables(contract.Descriptor{Migrations: tt.fsys})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
