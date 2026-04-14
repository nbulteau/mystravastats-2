package statistics

import (
	"encoding/json"
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
)

type bestEffortCacheKey struct {
	ActivityID int64  `json:"activityId"`
	Metric     string `json:"metric"`
	Target     string `json:"target"`
	StreamSize int    `json:"streamSize"`
}

type bestEffortCacheValue struct {
	HasValue bool                     `json:"hasValue"`
	Effort   *business.ActivityEffort `json:"effort,omitempty"`
}

type persistedBestEffortEntry struct {
	Key   bestEffortCacheKey   `json:"key"`
	Value bestEffortCacheValue `json:"value"`
}

var (
	bestEffortCacheMutex sync.RWMutex
	bestEffortCache      = map[bestEffortCacheKey]bestEffortCacheValue{}
)

func getOrComputeBestEffort(
	activityID int64,
	metric string,
	target string,
	stream *strava.Stream,
	supplier func() *business.ActivityEffort,
) *business.ActivityEffort {
	if stream == nil {
		return supplier()
	}

	key := bestEffortCacheKey{
		ActivityID: activityID,
		Metric:     metric,
		Target:     target,
		StreamSize: len(stream.Distance.Data),
	}

	bestEffortCacheMutex.RLock()
	if cached, ok := bestEffortCache[key]; ok {
		bestEffortCacheMutex.RUnlock()
		if !cached.HasValue || cached.Effort == nil {
			return nil
		}
		return cloneActivityEffort(cached.Effort)
	}
	bestEffortCacheMutex.RUnlock()

	computed := supplier()
	value := bestEffortCacheValue{
		HasValue: computed != nil,
		Effort:   cloneActivityEffort(computed),
	}

	bestEffortCacheMutex.Lock()
	bestEffortCache[key] = value
	bestEffortCacheMutex.Unlock()

	return cloneActivityEffort(computed)
}

func LoadBestEffortCacheFromDisk(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read best effort cache: %w", err)
	}

	var persisted []persistedBestEffortEntry
	if err := json.Unmarshal(data, &persisted); err != nil {
		return 0, fmt.Errorf("unmarshal best effort cache: %w", err)
	}

	loaded := map[bestEffortCacheKey]bestEffortCacheValue{}
	for _, entry := range persisted {
		loaded[entry.Key] = bestEffortCacheValue{
			HasValue: entry.Value.HasValue,
			Effort:   cloneActivityEffort(entry.Value.Effort),
		}
	}

	bestEffortCacheMutex.Lock()
	bestEffortCache = loaded
	bestEffortCacheMutex.Unlock()

	return len(loaded), nil
}

func SaveBestEffortCacheToDisk(path string) (int, error) {
	bestEffortCacheMutex.RLock()
	persisted := make([]persistedBestEffortEntry, 0, len(bestEffortCache))
	for key, value := range bestEffortCache {
		persisted = append(persisted, persistedBestEffortEntry{
			Key: key,
			Value: bestEffortCacheValue{
				HasValue: value.HasValue,
				Effort:   cloneActivityEffort(value.Effort),
			},
		})
	}
	bestEffortCacheMutex.RUnlock()

	sort.Slice(persisted, func(i, j int) bool {
		left := persisted[i].Key
		right := persisted[j].Key
		if left.ActivityID != right.ActivityID {
			return left.ActivityID < right.ActivityID
		}
		if left.Metric != right.Metric {
			return left.Metric < right.Metric
		}
		if left.Target != right.Target {
			return left.Target < right.Target
		}
		return left.StreamSize < right.StreamSize
	})

	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return 0, fmt.Errorf("create best effort cache dir: %w", err)
	}

	payload, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("marshal best effort cache: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return 0, fmt.Errorf("write temp best effort cache: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return 0, fmt.Errorf("rename best effort cache: %w", err)
	}

	return len(persisted), nil
}

func InvalidateBestEffortCacheByActivityIDs(activityIDs map[int64]struct{}) int {
	if len(activityIDs) == 0 {
		return 0
	}

	bestEffortCacheMutex.Lock()
	defer bestEffortCacheMutex.Unlock()

	removed := 0
	for key := range bestEffortCache {
		if _, matched := activityIDs[key.ActivityID]; matched {
			delete(bestEffortCache, key)
			removed++
		}
	}

	return removed
}

func ClearBestEffortCache() {
	bestEffortCacheMutex.Lock()
	bestEffortCache = map[bestEffortCacheKey]bestEffortCacheValue{}
	bestEffortCacheMutex.Unlock()
}

func BestEffortCacheSize() int {
	bestEffortCacheMutex.RLock()
	defer bestEffortCacheMutex.RUnlock()
	return len(bestEffortCache)
}

func effortDistanceTarget(distance float64) string {
	return strconv.FormatFloat(distance, 'f', -1, 64)
}

func effortSecondsTarget(seconds int) string {
	return strconv.Itoa(seconds)
}

func EffortDistanceTarget(distance float64) string {
	return effortDistanceTarget(distance)
}

func EffortSecondsTarget(seconds int) string {
	return effortSecondsTarget(seconds)
}

func cloneActivityEffort(source *business.ActivityEffort) *business.ActivityEffort {
	if source == nil {
		return nil
	}

	cloned := *source
	if source.AveragePower != nil {
		value := *source.AveragePower
		cloned.AveragePower = &value
	}
	return &cloned
}
