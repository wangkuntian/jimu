package main

import (
	"fmt"
	"os"

	"jimu/internal/assembly"
	"jimu/internal/profiles/minimal"
)

func main() {
	if err := assembly.Run(minimal.Assembly()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
