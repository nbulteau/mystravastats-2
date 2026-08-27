package infrastructure

import (
	"os"
	"strings"
)

func (adapter *OSMRoutingAdapter) detectExtractProfile() string {
	if normalized := normalizeOSRMProfile(strings.TrimSpace(adapter.extractProfileEnv)); normalized != "" {
		return normalized
	}
	for _, candidatePath := range adapter.profileMarkerCandidatePaths() {
		if normalized := normalizeOSRMProfile(readFirstLine(candidatePath)); normalized != "" {
			return normalized
		}
	}
	if normalized := normalizeOSRMProfile(strings.TrimSpace(adapter.profileOverride)); normalized != "" {
		return normalized
	}
	return "unknown"
}

func (adapter *OSMRoutingAdapter) profileMarkerCandidatePaths() []string {
	rawCandidates := []string{
		strings.TrimSpace(adapter.extractProfileCfgFile),
		defaultOSRMProfileFilePath,
		fallbackOSRMProfilePath,
	}
	seen := map[string]struct{}{}
	candidates := make([]string, 0, len(rawCandidates))
	for _, rawPath := range rawCandidates {
		cleanPath := strings.TrimSpace(rawPath)
		if cleanPath == "" {
			continue
		}
		if _, alreadyExists := seen[cleanPath]; alreadyExists {
			continue
		}
		seen[cleanPath] = struct{}{}
		candidates = append(candidates, cleanPath)
	}
	return candidates
}

func (adapter *OSMRoutingAdapter) effectiveRoutingProfile(extractProfile string) string {
	if normalized := normalizeOSRMProfile(strings.TrimSpace(adapter.profileOverride)); normalized != "" && normalized != "unknown" {
		switch normalized {
		case "/opt/bicycle.lua":
			return "cycling"
		case "/opt/foot.lua":
			return "walking"
		case "/opt/car.lua":
			return "driving"
		}
	}
	switch extractProfile {
	case "/opt/bicycle.lua":
		return "cycling"
	case "/opt/foot.lua":
		return "walking"
	case "/opt/car.lua":
		return "driving"
	default:
		// Conservative default for this product: cycling is the primary OSRM mode.
		return "cycling"
	}
}

func supportedRouteTypesByProfile(extractProfile string, effectiveProfile string) []string {
	switch strings.TrimSpace(strings.ToLower(effectiveProfile)) {
	case "cycling":
		return []string{"RIDE", "MTB", "GRAVEL"}
	case "walking":
		return []string{"RUN", "TRAIL", "HIKE"}
	case "driving":
		return []string{"RIDE"}
	default:
		return supportedRouteTypesByExtractProfile(extractProfile)
	}
}

func normalizeOSRMProfile(raw string) string {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	switch {
	case normalized == "":
		return ""
	case strings.Contains(normalized, "bicycle.lua"), normalized == "cycling":
		return "/opt/bicycle.lua"
	case strings.Contains(normalized, "foot.lua"), normalized == "walking":
		return "/opt/foot.lua"
	case strings.Contains(normalized, "car.lua"), normalized == "driving":
		return "/opt/car.lua"
	default:
		return "unknown"
	}
}

func supportedRouteTypesByExtractProfile(extractProfile string) []string {
	switch extractProfile {
	case "/opt/bicycle.lua":
		return []string{"RIDE", "MTB", "GRAVEL"}
	case "/opt/foot.lua":
		return []string{"RUN", "TRAIL", "HIKE"}
	case "/opt/car.lua":
		return []string{"RIDE"}
	default:
		return []string{"RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE"}
	}
}

func readFirstLine(path string) string {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return ""
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[0])
}
