package main

import (
	"fmt"
	"os"

	"jimu/internal/assembly"
	"jimu/internal/profiles/enterprise"
)

func main() {
	if err := assembly.Run(enterprise.Assembly()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
