package localrepository

import (
	"encoding/json"
	"fmt"
	"log"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type StravaRepository struct {
	cacheDirectory string
}

func NewStravaRepository(stravaCache string) *StravaRepository {
	return &StravaRepository{
		cacheDirectory: stravaCache,
	}
}

func (repo *StravaRepository) InitLocalStorageForClientId(clientId string) {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	if _, err := os.Stat(activitiesDirectory); os.IsNotExist(err) {
		_ = os.MkdirAll(activitiesDirectory, os.ModePerm)
	}
}

func (repo *StravaRepository) LoadAthleteFromCache(clientId string) strava.Athlete {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	athleteJsonFile := filepath.Join(activitiesDirectory, fmt.Sprintf("athlete-%s.json", clientId))

	if _, err := os.Stat(athleteJsonFile); os.IsNotExist(err) {
		log.Printf("No stravaAthlete found in cache")
		return fallbackAthlete(clientId)
	}

	data, err := os.ReadFile(athleteJsonFile)
	if err != nil {
		log.Printf("Failed to read athlete file '%s': %v", athleteJsonFile, err)
		return fallbackAthlete(clientId)
	}

	var athlete strava.Athlete
	if err := json.Unmarshal(data, &athlete); err != nil {
		log.Printf("Failed to unmarshal athlete data from '%s': %v", athleteJsonFile, err)
		return fallbackAthlete(clientId)
	}

	return athlete
}

func (repo *StravaRepository) SaveAthleteToCache(clientId string, stravaAthlete strava.Athlete) {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	_ = os.MkdirAll(activitiesDirectory, os.ModePerm)
	athleteJsonFile := filepath.Join(activitiesDirectory, fmt.Sprintf("athlete-%s.json", clientId))

	data, err := json.MarshalIndent(stravaAthlete, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal athlete data for clientId=%s: %v", clientId, err)
		return
	}

	if err := os.WriteFile(athleteJsonFile, data, os.ModePerm); err != nil {
		log.Printf("Failed to write athlete file '%s': %v", athleteJsonFile, err)
		return
	}
}

func (repo *StravaRepository) LoadActivitiesFromCache(clientId string, year int) []strava.Activity {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	yearActivitiesJsonFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("activities-%s-%d.json", clientId, year))

	if _, err := os.Stat(yearActivitiesJsonFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(yearActivitiesJsonFile)
	if err != nil {
		log.Printf("Failed to read activities file '%s': %v", yearActivitiesJsonFile, err)
		return nil
	}

	var activities []strava.Activity
	if err := json.Unmarshal(data, &activities); err != nil {
		log.Printf("Failed to unmarshal activities from '%s': %v", yearActivitiesJsonFile, err)
		return nil
	}

	return activities
}

func (repo *StravaRepository) IsLocalCacheExistForYear(clientId string, year int) bool {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	yearActivitiesJsonFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("activities-%s-%d.json", clientId, year))

	return fileExists(yearActivitiesJsonFile)
}

// GetLocalCacheLastModified returns the last modified date of the cache file for the given year.
func (repo *StravaRepository) GetLocalCacheLastModified(clientId string, year int) int64 {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	yearActivitiesJsonFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("activities-%s-%d.json", clientId, year))

	return lastModified(yearActivitiesJsonFile)
}

func (repo *StravaRepository) SaveActivitiesToCache(clientId string, year int, activities []strava.Activity) {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	_ = os.MkdirAll(yearActivitiesDirectory, os.ModePerm)

	data, err := json.MarshalIndent(activities, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal activities for clientId=%s year=%d: %v", clientId, year, err)
		return
	}

	yearActivitiesJsonFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("activities-%s-%d.json", clientId, year))
	if err := os.WriteFile(yearActivitiesJsonFile, data, os.ModePerm); err != nil {
		log.Printf("Failed to write activities file '%s': %v", yearActivitiesJsonFile, err)
		return
	}
}

func (repo *StravaRepository) LoadDetailedActivityFromCache(clientId string, year int, activityId int64) *strava.DetailedActivity {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	detailedActivityFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("stravaActivity-%d", activityId))

	if _, err := os.Stat(detailedActivityFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(detailedActivityFile)
	if err != nil {
		log.Printf("Failed to read detailed activity file '%s': %v", detailedActivityFile, err)
		return nil
	}

	var detailedActivity strava.DetailedActivity
	if err := json.Unmarshal(data, &detailedActivity); err != nil {
		log.Printf("Failed to unmarshal detailed activity from '%s': %v", detailedActivityFile, err)
		return nil
	}

	return &detailedActivity
}

func (repo *StravaRepository) SaveDetailedActivityToCache(clientId string, year int, stravaDetailedActivity strava.DetailedActivity) {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	detailedActivityFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("stravaActivity-%d", stravaDetailedActivity.Id))

	data, err := json.Marshal(stravaDetailedActivity)
	if err != nil {
		log.Printf("Failed to marshal detailed activity id=%d: %v", stravaDetailedActivity.Id, err)
		return
	}

	if err := os.WriteFile(detailedActivityFile, data, os.ModePerm); err != nil {
		log.Printf("Failed to write detailed activity file '%s': %v", detailedActivityFile, err)
		return
	}
}

func (repo *StravaRepository) LoadActivitiesStreamsFromCache(clientId string, year int, stravaActivity strava.Activity) *strava.Stream {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	streamFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("stream-%d", stravaActivity.Id))

	if _, err := os.Stat(streamFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(streamFile)
	if err != nil {
		log.Printf("Failed to read stream file '%s': %v", streamFile, err)
		return nil
	}

	var stream strava.Stream
	if err := json.Unmarshal(data, &stream); err != nil {
		log.Printf("Failed to unmarshal stream '%s' data: %v", streamFile, err)
		return nil
	}

	return &stream
}

func (repo *StravaRepository) SaveActivitiesStreamsToCache(clientId string, year int, stravaActivity strava.Activity, stream strava.Stream) {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	streamFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("stream-%d", stravaActivity.Id))

	data, err := json.Marshal(stream)
	if err != nil {
		log.Printf("Failed to marshal stream for activityId=%d: %v", stravaActivity.Id, err)
		return
	}

	if err := os.WriteFile(streamFile, data, os.ModePerm); err != nil {
		log.Printf("Failed to write stream file '%s': %v", streamFile, err)
		return
	}
}

func (repo *StravaRepository) BuildStreamIdsSet(clientId string, year int) map[int64]bool {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))

	streamIdsSet := make(map[int64]bool)
	err := filepath.Walk(yearActivitiesDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), "stream-") {
			id := strings.TrimPrefix(info.Name(), "stream-")
			streamId, err := strconv.ParseInt(id, 10, 64)
			if err == nil {
				streamIdsSet[streamId] = true
			}
		}
		return nil
	})
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to build stream ids set in '%s': %v", yearActivitiesDirectory, err)
		}
		return streamIdsSet
	}

	return streamIdsSet
}

func (repo *StravaRepository) ReadStravaAuthentication(stravaCache string) (string, string, bool) {
	cacheDirectory := filepath.Join(stravaCache)
	file := filepath.Join(cacheDirectory, ".strava")
	properties := make(map[string]string)

	data, err := os.ReadFile(file)
	if err != nil {
		log.Printf("File .strava not found at '%s': %v", file, err)
		return "", "", false
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			properties[parts[0]] = parts[1]
		}
	}

	clientId := strings.TrimSpace(properties["clientId"])
	clientSecret := strings.TrimSpace(properties["clientSecret"])
	useCache := strings.TrimSpace(properties["useCache"]) == "true"

	return clientId, clientSecret, useCache
}

func (repo *StravaRepository) UpdateStravaAuthentication(stravaCache, clientId, clientSecret string, useCache bool) {
	cacheDirectory := filepath.Join(stravaCache)
	file := filepath.Join(cacheDirectory, ".strava")
	properties := make(map[string]string)

	// Load existing properties if file exists
	if data, err := os.ReadFile(file); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				properties[parts[0]] = parts[1]
			}
		}
	}

	// Update properties
	properties["clientId"] = clientId
	properties["clientSecret"] = clientSecret
	properties["useCache"] = fmt.Sprintf("%v", useCache)

	// Build content
	var content strings.Builder
	content.WriteString("#Strava authentication\n")
	content.WriteString(fmt.Sprintf("clientId=%s\n", properties["clientId"]))
	content.WriteString(fmt.Sprintf("clientSecret=%s\n", properties["clientSecret"]))
	content.WriteString(fmt.Sprintf("useCache=%s\n", properties["useCache"]))

	// Write to file
	if err := os.WriteFile(file, []byte(content.String()), 0644); err != nil {
		log.Printf("Failed to update Strava authentication file: %v", err)
	} else {
		log.Printf("Updated Strava authentication file: useCache=%v", useCache)
	}
}

func (repo *StravaRepository) LoadHeartRateZoneSettings(clientId string) business.HeartRateZoneSettings {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	settingsFile := filepath.Join(activitiesDirectory, fmt.Sprintf("heart-rate-zones-%s.json", clientId))
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		return business.HeartRateZoneSettings{}
	}

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		log.Printf("Failed to read heart rate zone settings file '%s': %v", settingsFile, err)
		return business.HeartRateZoneSettings{}
	}

	var settings business.HeartRateZoneSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		log.Printf("Failed to unmarshal heart rate zone settings from '%s': %v", settingsFile, err)
		return business.HeartRateZoneSettings{}
	}

	return settings
}

func (repo *StravaRepository) SaveHeartRateZoneSettings(clientId string, settings business.HeartRateZoneSettings) {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	if err := os.MkdirAll(activitiesDirectory, os.ModePerm); err != nil {
		log.Printf("Failed to create heart rate settings directory '%s': %v", activitiesDirectory, err)
		return
	}

	settingsFile := filepath.Join(activitiesDirectory, fmt.Sprintf("heart-rate-zones-%s.json", clientId))
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal heart rate zone settings for clientId=%s: %v", clientId, err)
		return
	}

	if err := os.WriteFile(settingsFile, data, os.ModePerm); err != nil {
		log.Printf("Failed to write heart rate zone settings file '%s': %v", settingsFile, err)
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func lastModified(filename string) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		return 0
	}

	return info.ModTime().UnixMilli()
}

func (repo *StravaRepository) loadActivitiesStreams(activities []strava.Activity, activitiesDirectory string) {
	for i, activity := range activities {
		streamFile := filepath.Join(activitiesDirectory, fmt.Sprintf("stream-%d", activity.Id))
		if _, err := os.Stat(streamFile); err == nil {
			data, err := os.ReadFile(streamFile)
			if err != nil {
				log.Printf("Failed to read stream file '%s': %v", streamFile, err)
				continue
			}
			var stream strava.Stream
			if err := json.Unmarshal(data, &stream); err != nil {
				log.Printf("Failed to unmarshal stream file '%s': %v", streamFile, err)
				continue
			}
			activities[i].Stream = &stream
		}
	}
}

func fallbackAthlete(clientId string) strava.Athlete {
	clientIdInt, err := strconv.ParseInt(clientId, 10, 64)
	if err != nil {
		log.Printf("Failed to convert clientId to int64 for fallback athlete: %v", err)
		clientIdInt = 0
	}
	username := ""
	return strava.Athlete{Id: clientIdInt, Username: &username}
}
