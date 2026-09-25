package factorio

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var modDependencyNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type parsedModDependency struct {
	Name            string
	Kind            string
	Operator        string
	RequiredVersion Version
	VersionText     string
	HasVersion      bool
}

type ModDependencyStatus struct {
	Raw              string `json:"raw"`
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	Operator         string `json:"operator,omitempty"`
	RequiredVersion  string `json:"required_version,omitempty"`
	InstalledVersion string `json:"installed_version,omitempty"`
	State            string `json:"state"`
	Satisfied        bool   `json:"satisfied"`
}

func parseModDependency(raw string) (parsedModDependency, error) {
	value := strings.TrimSpace(raw)
	dependency := parsedModDependency{Kind: "required"}
	if strings.HasPrefix(value, "(?)") {
		dependency.Kind = "hidden-optional"
		value = strings.TrimSpace(strings.TrimPrefix(value, "(?)"))
	} else if value != "" {
		switch value[0] {
		case '?':
			dependency.Kind = "optional"
			value = strings.TrimSpace(value[1:])
		case '!':
			dependency.Kind = "incompatible"
			value = strings.TrimSpace(value[1:])
		case '~':
			dependency.Kind = "no-load-order"
			value = strings.TrimSpace(value[1:])
		}
	}

	parts := strings.Fields(value)
	if len(parts) != 1 && len(parts) != 3 {
		return parsedModDependency{}, fmt.Errorf("invalid dependency %q", raw)
	}
	dependency.Name = parts[0]
	if !modDependencyNamePattern.MatchString(dependency.Name) {
		return parsedModDependency{}, fmt.Errorf("invalid dependency name %q", dependency.Name)
	}
	if len(parts) == 3 {
		dependency.Operator = parts[1]
		if dependency.Operator == "=" {
			dependency.Operator = "=="
		}
		switch dependency.Operator {
		case "==", "!=", ">", "<", ">=", "<=":
		default:
			return parsedModDependency{}, fmt.Errorf("unsupported dependency operator %q", parts[1])
		}
		if err := dependency.RequiredVersion.UnmarshalText([]byte(parts[2])); err != nil {
			return parsedModDependency{}, fmt.Errorf("invalid dependency version %q: %w", parts[2], err)
		}
		dependency.VersionText = parts[2]
		dependency.HasVersion = true
	}
	return dependency, nil
}

func (mods *Mods) applyDependencyStatus(result *ModsResultList) {
	versions := make(map[string]Version, len(mods.ModInfoList.Mods)+len(mods.ModSimpleList.Mods)+1)
	enabled := make(map[string]bool, len(mods.ModSimpleList.Mods)+1)
	server := GetFactorioServer()
	installedFactorioVersion := server.Version
	if server.BaseModVersion != "" {
		_ = installedFactorioVersion.UnmarshalText([]byte(server.BaseModVersion))
	}
	versions["base"] = installedFactorioVersion
	enabled["base"] = true
	versions["core"] = installedFactorioVersion
	enabled["core"] = true
	for _, simple := range mods.ModSimpleList.Mods {
		enabled[simple.Name] = simple.Enabled
		// Built-in expansion mods are listed in mod-list.json but do not live
		// in the writable mods directory. They follow the Factorio release.
		if _, exists := versions[simple.Name]; !exists {
			versions[simple.Name] = installedFactorioVersion
		}
	}
	for _, info := range mods.ModInfoList.Mods {
		var version Version
		if err := version.UnmarshalText([]byte(info.Version)); err == nil {
			versions[info.Name] = version
		}
	}

	for index := range result.ModsResult {
		mod := &result.ModsResult[index]
		mod.DependencyStatus = make([]ModDependencyStatus, 0, len(mod.Dependencies))
		mod.CompatibilityIssues = make([]string, 0)
		mod.Compatibility = mod.FactorioCompatible
		if !mod.FactorioCompatible {
			mod.CompatibilityIssues = append(mod.CompatibilityIssues, "factorio-version")
		}
		for _, raw := range mod.Dependencies {
			parsed, err := parseModDependency(raw)
			status := ModDependencyStatus{Raw: raw, State: "ok", Satisfied: true}
			if err != nil {
				status.State = "invalid"
				status.Satisfied = false
				mod.Compatibility = false
				mod.CompatibilityIssues = append(mod.CompatibilityIssues, "invalid-dependency")
				mod.DependencyStatus = append(mod.DependencyStatus, status)
				continue
			}
			status.Name = parsed.Name
			status.Kind = parsed.Kind
			status.Operator = parsed.Operator
			status.RequiredVersion = parsed.VersionText
			actual, installed := versions[parsed.Name]
			isEnabled := enabled[parsed.Name]
			if installed {
				status.InstalledVersion = strings.TrimSuffix(actual.String(), ".0")
			}

			switch parsed.Kind {
			case "incompatible":
				if installed && isEnabled && (!parsed.HasVersion || actual.Compatible(parsed.RequiredVersion, parsed.Operator)) {
					status.State = "conflict"
					status.Satisfied = false
				}
			case "optional", "hidden-optional":
				if installed && isEnabled && parsed.HasVersion && !actual.Compatible(parsed.RequiredVersion, parsed.Operator) {
					status.State = "version-mismatch"
					status.Satisfied = false
				}
			default:
				if !installed {
					status.State = "missing"
					status.Satisfied = false
				} else if !isEnabled {
					status.State = "disabled"
					status.Satisfied = false
				} else if parsed.HasVersion && !actual.Compatible(parsed.RequiredVersion, parsed.Operator) {
					status.State = "version-mismatch"
					status.Satisfied = false
				}
			}
			if !status.Satisfied {
				mod.Compatibility = false
				mod.CompatibilityIssues = append(mod.CompatibilityIssues, status.State+":"+status.Name)
			}
			mod.DependencyStatus = append(mod.DependencyStatus, status)
		}
	}
}

func (mods *Mods) EnabledRequiredBy(modName string) []string {
	enabled := make(map[string]bool, len(mods.ModSimpleList.Mods))
	for _, simple := range mods.ModSimpleList.Mods {
		enabled[simple.Name] = simple.Enabled
	}
	requiredBy := make([]string, 0)
	for _, info := range mods.ModInfoList.Mods {
		if !enabled[info.Name] || info.Name == modName {
			continue
		}
		for _, raw := range info.Dependencies {
			dependency, err := parseModDependency(raw)
			if err == nil && dependency.Name == modName && (dependency.Kind == "required" || dependency.Kind == "no-load-order") {
				requiredBy = append(requiredBy, info.Name)
				break
			}
		}
	}
	sort.Strings(requiredBy)
	return requiredBy
}
