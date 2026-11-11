//go:build !jimu_no_otel && !jimu_nodb

package app

import gormotel "gorm.io/plugin/opentelemetry/tracing"

func (s dbState) instrumentOTel() error {
	if s.conn == nil {
		return nil
	}
	return s.conn.Use(gormotel.NewPlugin())
}
