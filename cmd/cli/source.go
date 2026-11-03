package main

import (
	"errors"

	"jimu/internal/templatesource"
	"jimu/tools/generator"
)

var errFrameworkSourceRootUnavailable = errors.New("framework source root is unavailable; run from a Jimu checkout or pass --from")

func resolveFromCheckout(explicitRoot string) (templatesource.Source, func() error, error) {
	root := explicitRoot
	if root == "" {
		root = generator.FrameworkRoot()
		if root == "" {
			return templatesource.Source{}, func() error { return nil }, errFrameworkSourceRootUnavailable
		}
	}
	return templatesource.Source{Root: root}, func() error { return nil }, nil
}
