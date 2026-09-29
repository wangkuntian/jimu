package ws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientCanSubscribe(t *testing.T) {
	c := &Client{userID: 42}

	tests := []struct {
		name    string
		channel string
		want    bool
	}{
		{"own user channel", "user:42", true},
		{"other user channel", "user:7", false},
		{"malformed user channel", "user:abc", false},
		{"public room channel", "room:abc", true},
		{"broadcast channel", "broadcast", true},
		{"room with own id suffix", "room:42", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, c.canSubscribe(tt.channel))
		})
	}
}
