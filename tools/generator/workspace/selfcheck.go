package workspace

import (
	"os"
	"path/filepath"
)

func SelfCheck(dir string) error {
	if err := runGo(dir, []string{"GOWORK=off"}, "build", "./..."); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, "tools", "checkcapabilities")); err != nil {
		if os.IsNotExist(err) {
			return runGo(dir, []string{"GOWORK=off"}, "test", "./...")
		}
		return err
	}
	return runGo(dir, []string{"GOWORK=off"}, "run", "./tools/checkcapabilities")
}
