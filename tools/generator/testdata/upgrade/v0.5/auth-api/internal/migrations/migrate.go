package migrations

import (
	"embed"
	"io/fs"
	"strings"
)

//go:embed mysql/*.sql
var files embed.FS

func Files() ([]string, error) {
	var out []string
	err := fs.WalkDir(files, "mysql", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".sql") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func Read(name string) (string, error) {
	content, err := files.ReadFile(name)
	return string(content), err
}
