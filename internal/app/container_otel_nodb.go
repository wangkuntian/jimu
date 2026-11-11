//go:build !jimu_no_otel && jimu_nodb

package app

func (dbState) instrumentOTel() error { return nil }
