package main

import (
	"fmt"
	"os"

	"example.com/upgrade-auth-api/internal/app"
)

var version = "dev"

func main() {
	if err := app.Run(version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
