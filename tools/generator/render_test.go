package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTextGoesThroughGofmt(t *testing.T) {
	out, err := RenderText("x.go.tmpl", "package  x\n\nvar   A=1\n", nil)
	require.NoError(t, err)
	assert.Equal(t, "package x\n\nvar A = 1\n", string(out))
}

func TestTemplateNamesCoversEveryEmbeddedFile(t *testing.T) {
	names := TemplateNames()
	require.NotEmpty(t, names)
	for _, n := range names {
		_, err := Template(n)
		require.NoErrorf(t, err, "template %q", n)
	}
}
