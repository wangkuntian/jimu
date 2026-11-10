package db

import (
	"testing"

	"jimu/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestPostgresDSNPinsUTC(t *testing.T) {
	got := pgDSNHost(config.DBConfig{User: "u", Password: "p", Host: "h", Port: 5432, Database: "d"}, "", 0)
	assert.Contains(t, got, "TimeZone=UTC")
}
