package workspace

func SelfCheck(dir string) error {
	if err := runGo(dir, []string{"GOWORK=off"}, "build", "./..."); err != nil {
		return err
	}
	return runGo(dir, []string{"GOWORK=off"}, "run", "./tools/checkcapabilities")
}
