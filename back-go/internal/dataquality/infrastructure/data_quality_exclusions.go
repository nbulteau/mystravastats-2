package infrastructure

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

const (
	dataQualitySecureDirMode  = 0700
	dataQualitySecureFileMode = 0600
)

func CurrentProviderExclusions() map[int64]business.DataQualityExclusion {
	provider := activityprovider.Get()
	exclusions := loadExclusions(provider.CacheRootPath(), provider.ClientID())
	return exclusionsByActivityID(exclusions)
}

func FilterExcludedFromStats(activities []*strava.Activity) []*strava.Activity {
	if len(activities) == 0 {
		return []*strava.Activity{}
	}

	exclusions := CurrentProviderExclusions()
	if len(exclusions) == 0 {
		return cloneDataQualityActivityPointers(activities)
	}

	filtered := make([]*strava.Activity, 0, len(activities))
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		if _, excluded := exclusions[activity.Id]; excluded {
			continue
		}
		filtered = append(filtered, activity)
	}
	return filtered
}

func cloneDataQualityActivityPointers(activities []*strava.Activity) []*strava.Activity {
	cloned := make([]*strava.Activity, len(activities))
	copy(cloned, activities)
	return cloned
}

func ExclusionSignature() string {
	provider := activityprovider.Get()
	exclusionsFile := exclusionsFilePath(provider.CacheRootPath(), provider.ClientID())
	info, err := os.Stat(exclusionsFile)
	if err != nil {
		return "none"
	}
	return fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), info.Size())
}

func excludeCurrentProviderActivityFromStats(activityID int64, reason string) (business.DataQualityReport, error) {
	if activityID <= 0 {
		return business.DataQualityReport{}, fmt.Errorf("activityId must be > 0")
	}

	provider := activityprovider.Get()
	activities := provider.GetActivitiesByYearAndActivityTypes(nil, allActivityTypes()...)
	activity := findActivity(activities, activityID)
	if activity == nil {
		return business.DataQualityReport{}, fmt.Errorf("activity %d not found", activityID)
	}

	exclusions := loadExclusions(provider.CacheRootPath(), provider.ClientID())
	exclusionsByID := exclusionsByActivityID(exclusions)
	source := currentProviderName(provider)
	normalizedReason := strings.TrimSpace(reason)
	if normalizedReason == "" {
		normalizedReason = "Excluded from statistics after data quality audit."
	}

	exclusionsByID[activityID] = business.DataQualityExclusion{
		ActivityID:   activityID,
		Source:       strings.ToUpper(source),
		ActivityName: strings.TrimSpace(activity.Name),
		ActivityType: activity.Type,
		Year:         extractIssueYear(activity),
		Reason:       normalizedReason,
		ExcludedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	if err := saveExclusions(provider.CacheRootPath(), provider.ClientID(), sortedExclusions(exclusionsByID)); err != nil {
		return business.DataQualityReport{}, err
	}
	return CurrentProviderReport(), nil
}

func includeCurrentProviderActivityInStats(activityID int64) (business.DataQualityReport, error) {
	if activityID <= 0 {
		return business.DataQualityReport{}, fmt.Errorf("activityId must be > 0")
	}

	provider := activityprovider.Get()
	exclusionsByID := exclusionsByActivityID(loadExclusions(provider.CacheRootPath(), provider.ClientID()))
	delete(exclusionsByID, activityID)
	if err := saveExclusions(provider.CacheRootPath(), provider.ClientID(), sortedExclusions(exclusionsByID)); err != nil {
		return business.DataQualityReport{}, err
	}
	return CurrentProviderReport(), nil
}

func loadExclusions(cacheRoot string, clientID string) []business.DataQualityExclusion {
	exclusionsFile := exclusionsFilePath(cacheRoot, clientID)
	data, err := os.ReadFile(exclusionsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Unable to read data quality exclusions from %s: %v", exclusionsFile, err)
		}
		return []business.DataQualityExclusion{}
	}

	var payload struct {
		Exclusions []business.DataQualityExclusion `json:"exclusions"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		log.Printf("Unable to parse data quality exclusions from %s: %v", exclusionsFile, err)
		return []business.DataQualityExclusion{}
	}
	return payload.Exclusions
}

func saveExclusions(cacheRoot string, clientID string, exclusions []business.DataQualityExclusion) error {
	athleteDirectory := exclusionsDirectory(cacheRoot, clientID)
	if err := os.MkdirAll(athleteDirectory, dataQualitySecureDirMode); err != nil {
		return fmt.Errorf("unable to create data quality directory: %w", err)
	}

	payload := struct {
		Exclusions []business.DataQualityExclusion `json:"exclusions"`
	}{
		Exclusions: exclusions,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to encode data quality exclusions: %w", err)
	}
	if err := os.WriteFile(exclusionsFilePath(cacheRoot, clientID), data, dataQualitySecureFileMode); err != nil {
		return fmt.Errorf("unable to write data quality exclusions: %w", err)
	}
	return nil
}

func exclusionsDirectory(cacheRoot string, clientID string) string {
	return filepath.Join(cacheRoot, fmt.Sprintf("strava-%s", clientID))
}

func exclusionsFilePath(cacheRoot string, clientID string) string {
	return filepath.Join(exclusionsDirectory(cacheRoot, clientID), fmt.Sprintf("data-quality-exclusions-%s.json", clientID))
}

func exclusionsByActivityID(exclusions []business.DataQualityExclusion) map[int64]business.DataQualityExclusion {
	result := make(map[int64]business.DataQualityExclusion, len(exclusions))
	for _, exclusion := range exclusions {
		if exclusion.ActivityID <= 0 {
			continue
		}
		result[exclusion.ActivityID] = exclusion
	}
	return result
}

func sortedExclusions(exclusions map[int64]business.DataQualityExclusion) []business.DataQualityExclusion {
	result := make([]business.DataQualityExclusion, 0, len(exclusions))
	for _, exclusion := range exclusions {
		result = append(result, exclusion)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Year != result[j].Year {
			return result[i].Year > result[j].Year
		}
		return result[i].ActivityID < result[j].ActivityID
	})
	return result
}

func findActivity(activities []*strava.Activity, activityID int64) *strava.Activity {
	for _, activity := range activities {
		if activity != nil && activity.Id == activityID {
			return activity
		}
	}
	return nil
}

func currentProviderName(provider activityprovider.ActivityProvider) string {
	diagnostics := provider.CacheDiagnostics()
	return strings.ToLower(strings.TrimSpace(fmt.Sprint(diagnostics["provider"])))
}
