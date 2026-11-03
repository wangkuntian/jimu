package scaffold

import (
	"bytes"
	"testing"

	"jimu/internal/templatesource"
	"jimu/tools/generator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaffoldCommandUsesResolvedSource(t *testing.T) {
	root := generator.FrameworkRoot()
	require.NotEmpty(t, root)
	t.Chdir(t.TempDir())

	called := 0
	cleaned := 0
	resolve := func(explicitRoot string) (templatesource.Source, func() error, error) {
		called++
		return templatesource.Source{Root: root}, func() error {
			cleaned++
			return nil
		}, nil
	}

	cmd := NewCommand(resolve)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"demo", "--template=basic-api", "--dry-run"})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, 1, called)
	assert.Equal(t, 1, cleaned)
	assert.Contains(t, out.String(), "template     basic-api")
}
