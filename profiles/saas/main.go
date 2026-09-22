package main

import (
	"fmt"
	"os"

	"jimu/internal/assembly"
	"jimu/internal/profiles/saas"
)

func main() {
	if err := assembly.Run(saas.Assembly()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
