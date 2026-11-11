package profileassets

// GeneratedSelection lists the asset files copied into a generated project.
// The framework repository leaves it empty; the generator overwrites this file
// with the selected file set so generated-project gates validate only what was
// actually copied.
var GeneratedSelection []string

// SelectedFiles returns the generated asset selection, if any.
func SelectedFiles() []string {
	return append([]string(nil), GeneratedSelection...)
}
