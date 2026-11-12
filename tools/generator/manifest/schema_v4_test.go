package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func v3MinimalDocument() Document {
	doc := validDocument()
	doc.SchemaVersion = PreviousSchemaVersion
	doc.Generation = nil
	doc.Selection.Profile = "minimal"
	doc.Selection.Capabilities = []string{}
	doc.Selection.Drivers = map[string][]string{}
	doc.Composition.Requested = []string{}
	doc.Composition.Resolved = []string{}
	doc.Composition.AutoAdded = []string{}
	doc.Composition.Drivers = map[string][]string{}
	doc.Capabilities = []Capability{}
	doc.Report.Capabilities = []string{}
	doc.Digest = ""
	return doc
}

func writeV3MinimalFixture(t *testing.T) string {
	t.Helper()
	doc := v3MinimalDocument()
	content, err := json.Marshal(normalize(doc))
	require.NoError(t, err)
	sum := sha256.Sum256(content)
	doc.Digest = "sha256:" + hex.EncodeToString(sum[:])
	content, err = json.Marshal(normalize(doc))
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "v3-minimal.json")
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func TestV3MinimalGetsLegacyGenerationDefaults(t *testing.T) {
	doc, err := Load(writeV3MinimalFixture(t))
	require.NoError(t, err)
	require.Equal(t, CurrentSchemaVersion, doc.SchemaVersion)
	require.NotNil(t, doc.Generation)
	require.False(t, doc.Generation.RuntimeClosure)
	require.Equal(t, GenerationTestsAll, doc.Generation.Tests)
	require.Equal(t, GenerationToolsAll, doc.Generation.ToolsMode)
	require.Empty(t, doc.Generation.Deploy)
	require.Equal(t, GenerationCLIMinimal, doc.Generation.CLI)
	require.Equal(t, GenerationCLIUIDirect, doc.Generation.CLIUI)
}

func TestV4RequiresGenerationFields(t *testing.T) {
	doc := validDocument()
	doc.Generation = nil
	require.ErrorContains(t, Validate(doc), "generation")
}

func TestGenerationValidationRejectsUnknownModes(t *testing.T) {
	doc := validDocument()
	doc.Generation.Tests = "sometimes"
	require.ErrorContains(t, Validate(doc), "generation.tests")
}
