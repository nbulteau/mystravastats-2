package localrepository

import (
	"encoding/json"
	"fmt"
	"log"
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

		clientIdInt, err := strconv.ParseInt(clientId, 10, 64)
		if err != nil {
			log.Fatalf("Failed to convert clientId to int64: %v", err)
		}

		return strava.Athlete{Id: clientIdInt, Username: new(string)}
	}

	data, err := os.ReadFile(athleteJsonFile)
	if err != nil {
		log.Fatalf("Failed to read athlete file: %v", err)
	}

	var athlete strava.Athlete
	if err := json.Unmarshal(data, &athlete); err != nil {
		log.Fatalf("Failed to unmarshal athlete data: %v", err)
	}

	return athlete
}

func (repo *StravaRepository) SaveAthleteToCache(clientId string, stravaAthlete strava.Athlete) {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	_ = os.MkdirAll(activitiesDirectory, os.ModePerm)
	athleteJsonFile := filepath.Join(activitiesDirectory, fmt.Sprintf("athlete-%s.json", clientId))

	data, err := json.MarshalIndent(stravaAthlete, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal athlete data: %v", err)
	}

	if err := os.WriteFile(athleteJsonFile, data, os.ModePerm); err != nil {
		log.Fatalf("Failed to write athlete file: %v", err)
	}
}

func (repo *StravaRepository) LoadActivitiesFromCache(clientId string, year int) []strava.Activity {
	log.Printf("⌛ Load activities from cache for year %d", year)

	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	yearActivitiesJsonFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("activities-%s-%d.json", clientId, year))

	if _, err := os.Stat(yearActivitiesJsonFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(yearActivitiesJsonFile)
	if err != nil {
		log.Fatalf("Failed to read activities file: %v", err)
	}

	var activities []strava.Activity
	if err := json.Unmarshal(data, &activities); err != nil {
		log.Fatalf("Failed to unmarshal activities data: %v", err)
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
		log.Fatalf("Failed to marshal activities data: %v", err)
	}

	yearActivitiesJsonFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("activities-%s-%d.json", clientId, year))
	if err := os.WriteFile(yearActivitiesJsonFile, data, os.ModePerm); err != nil {
		log.Fatalf("Failed to write activities file: %v", err)
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
		log.Fatalf("Failed to read detailed activity file: %v", err)
	}

	var detailedActivity strava.DetailedActivity
	if err := json.Unmarshal(data, &detailedActivity); err != nil {
		log.Fatalf("Failed to unmarshal detailed activity data: %v", err)
	}

	return &detailedActivity
}

func (repo *StravaRepository) SaveDetailedActivityToCache(clientId string, year int, stravaDetailedActivity strava.DetailedActivity) {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	detailedActivityFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("stravaActivity-%d", stravaDetailedActivity.Id))

	data, err := json.Marshal(stravaDetailedActivity)
	if err != nil {
		log.Fatalf("Failed to marshal detailed activity data: %v", err)
	}

	if err := os.WriteFile(detailedActivityFile, data, os.ModePerm); err != nil {
		log.Fatalf("Failed to write detailed activity file: %v", err)
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
		log.Fatalf("Failed to read stream file '%s': %v", streamFile, err)
	}

	var stream strava.Stream
	if err := json.Unmarshal(data, &stream); err != nil {
		log.Fatalf("Failed to unmarshal stream '%s' data: %v", streamFile, err)
	}

	return &stream
}

func (repo *StravaRepository) SaveActivitiesStreamsToCache(clientId string, year int, stravaActivity strava.Activity, stream strava.Stream) {
	activitiesDirectory := filepath.Join(repo.cacheDirectory, fmt.Sprintf("strava-%s", clientId))
	yearActivitiesDirectory := filepath.Join(activitiesDirectory, fmt.Sprintf("strava-%s-%d", clientId, year))
	streamFile := filepath.Join(yearActivitiesDirectory, fmt.Sprintf("stream-%d", stravaActivity.Id))

	data, err := json.Marshal(stream)
	if err != nil {
		log.Fatalf("Failed to marshal stream data: %v", err)
	}

	if err := os.WriteFile(streamFile, data, os.ModePerm); err != nil {
		log.Fatalf("Failed to write stream file: %v", err)
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
		log.Fatalf("Failed to build stream ids set: %v", err)
	}

	return streamIdsSet
}

func (repo *StravaRepository) ReadStravaAuthentication(stravaCache string) (string, string, bool) {
	cacheDirectory := filepath.Join(stravaCache)
	file := filepath.Join(cacheDirectory, ".strava")
	properties := make(map[string]string)

	data, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("File .strava not found: %v", err)
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
				log.Fatalf("Failed to read stream file: %v", err)
			}
			var stream strava.Stream
			if err := json.Unmarshal(data, &stream); err != nil {
				log.Fatalf("Failed to unmarshal stream data: %v", err)
			}
			activities[i].Stream = &stream
		}
	}
}
