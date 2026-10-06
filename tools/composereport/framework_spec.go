package main

import (
	"jimu/tools/generator/frameworkmanifest"
	"jimu/tools/internal/projectmetrics"
)

func projectReportSpec(root, name string) (projectmetrics.ReportSpec, error) {
	doc, err := frameworkmanifest.Export(frameworkmanifest.Request{Root: root, Profile: name, Module: modulePath})
	if err != nil {
		return projectmetrics.ReportSpec{}, err
	}
	return projectmetrics.ReportSpec{
		Name:         name,
		Capabilities: doc.Report.Capabilities,
		Routes:       doc.Report.Routes,
		Migrations:   len(doc.Report.Migrations),
		Tables:       len(doc.Report.Tables),
	}, nil
}
