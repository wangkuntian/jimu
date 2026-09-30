package plan

import (
	"fmt"
	"slices"
	"sort"

	"jimu/tools/generator/manifest"
)

func Build(doc manifest.Document) (Plan, error) {
	if err := manifest.Validate(doc); err != nil {
		return Plan{}, err
	}
	if err := validateDestinations(doc); err != nil {
		return Plan{}, err
	}
	value := Plan{
		Selection:      normalizeSelection(doc.Selection),
		Copy:           normalizeCopy(doc.Copy),
		Templates:      normalizeTemplates(doc.Templates),
		Merges:         normalizeMerges(doc.Merges),
		Rewrites:       normalizeRewrites(doc.Rewrites),
		Assets:         normalizeAssets(doc.Assets),
		Prune:          normalizePrune(doc.Prune),
		Report:         normalizeReport(doc.Report),
		GeneratedFiles: sortedClone(doc.GeneratedFiles),
	}
	if err := Validate(value); err != nil {
		return Plan{}, err
	}
	return value, nil
}

func validateDestinations(doc manifest.Document) error {
	seen := map[string]string{}
	reserve := func(destination, kind string) error {
		if previous, ok := seen[destination]; ok {
			return fmt.Errorf("duplicate destination %q in %s and %s", destination, previous, kind)
		}
		seen[destination] = kind
		return nil
	}
	for _, action := range doc.Copy {
		if err := reserve(action.Destination, "copy"); err != nil {
			return err
		}
	}
	for _, action := range doc.Templates {
		if err := reserve(action.Destination, "templates"); err != nil {
			return err
		}
	}
	for _, action := range doc.Assets {
		if err := reserve(action.Destination, "assets"); err != nil {
			return err
		}
	}
	return nil
}

func normalizeSelection(value manifest.Selection) manifest.Selection {
	value.Capabilities = slices.Clone(value.Capabilities)
	if value.Drivers == nil {
		value.Drivers = map[string][]string{}
		return value
	}
	drivers := make(map[string][]string, len(value.Drivers))
	for name, selected := range value.Drivers {
		drivers[name] = sortedClone(selected)
	}
	value.Drivers = drivers
	return value
}

func normalizeCopy(values []manifest.CopyAction) []manifest.CopyAction {
	values = slices.Clone(values)
	for i := range values {
		values[i].Include = sortedClone(values[i].Include)
		values[i].Exclude = sortedClone(values[i].Exclude)
	}
	sort.SliceStable(values, func(i, j int) bool {
		return values[i].Destination < values[j].Destination
	})
	return values
}

func normalizeTemplates(values []manifest.TemplateAction) []manifest.TemplateAction {
	values = slices.Clone(values)
	for i := range values {
		if values[i].Data == nil {
			values[i].Data = map[string]string{}
		}
	}
	sort.SliceStable(values, func(i, j int) bool { return values[i].Destination < values[j].Destination })
	return values
}

func normalizeMerges(values []manifest.MergeAction) []manifest.MergeAction {
	values = slices.Clone(values)
	for i := range values {
		values[i].Sections = sortedClone(values[i].Sections)
	}
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Destination != values[j].Destination {
			return values[i].Destination < values[j].Destination
		}
		return values[i].Source < values[j].Source
	})
	return values
}

func normalizeRewrites(values []manifest.RewriteAction) []manifest.RewriteAction {
	values = slices.Clone(values)
	for i := range values {
		values[i].Files = sortedClone(values[i].Files)
	}
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Kind != values[j].Kind {
			return values[i].Kind < values[j].Kind
		}
		return values[i].From < values[j].From
	})
	return values
}

func normalizeAssets(values []manifest.AssetAction) []manifest.AssetAction {
	values = slices.Clone(values)
	for i := range values {
		values[i].Include = sortedClone(values[i].Include)
		values[i].Exclude = sortedClone(values[i].Exclude)
	}
	sort.SliceStable(values, func(i, j int) bool { return values[i].Destination < values[j].Destination })
	return values
}

func normalizePrune(values []manifest.PruneRule) []manifest.PruneRule {
	values = slices.Clone(values)
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Path != values[j].Path {
			return values[i].Path < values[j].Path
		}
		return values[i].Kind < values[j].Kind
	})
	return values
}

func normalizeReport(value manifest.ReportSpec) manifest.ReportSpec {
	value.Capabilities = slices.Clone(value.Capabilities)
	value.Migrations = sortedClone(value.Migrations)
	value.Tables = sortedClone(value.Tables)
	value.HeavyDeps = sortedClone(value.HeavyDeps)
	return value
}

func sortedClone(values []string) []string {
	if values == nil {
		return []string{}
	}
	out := slices.Clone(values)
	sort.Strings(out)
	return out
}
