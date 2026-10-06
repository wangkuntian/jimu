package plan

import "fmt"

func Validate(value Plan) error {
	if value.Selection.Shape == "" {
		return fmt.Errorf("plan selection shape is required")
	}
	if len(value.GeneratedFiles) == 0 {
		return fmt.Errorf("plan generated files are required")
	}
	return nil
}
