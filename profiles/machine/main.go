package main

import (
	"fmt"
	"os"

	"jimu/internal/assembly"
	"jimu/internal/profiles/machine"
)

func main() {
	if err := assembly.Run(machine.Assembly()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
