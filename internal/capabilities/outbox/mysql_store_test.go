package outbox

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMySQLStore_AddWithNilTx(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Event{}))

	s := NewMySQLStore(db)
	err = s.Add(context.Background(), nil, Event{
		AggregateID: "user:1",
		EventType:   "user.created",
		Payload:     []byte(`{"user_id":1}`),
	})
	require.NoError(t, err)

	var events []Event
	require.NoError(t, db.Find(&events).Error)
	require.Len(t, events, 1)
	assert.Equal(t, "user:1", events[0].AggregateID)
}

func TestMySQLStore_AddWithinTx(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Event{}))

	s := NewMySQLStore(db)
	err = db.Transaction(func(tx *gorm.DB) error {
		return s.Add(context.Background(), tx, Event{
			AggregateID: "user:2",
			EventType:   "user.updated",
			Payload:     []byte(`{"user_id":2}`),
		})
	})
	require.NoError(t, err)

	var events []Event
	require.NoError(t, db.Find(&events).Error)
	require.Len(t, events, 1)
	assert.Equal(t, "user.updated", events[0].EventType)
}
