package services

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	segmentAnalysisCacheSchemaVersion = 1
	segmentAnalysisCacheFileName      = "segment-analysis-cache-v1.json"
	segmentAnalysisCacheMaxEntries    = 256
	segmentAnalysisCacheTTL           = 30 * time.Minute
	segmentAnalysisFallbackCacheTTL   = 45 * time.Second
	segmentAnalysisAlgorithmVersion   = "direction-v2"
)

var segmentAttemptsCache = struct {
	sync.RWMutex
	loaded  bool
	entries map[string]segmentAttemptsCacheEntry
}{
	entries: make(map[string]segmentAttemptsCacheEntry),
}

type segmentAttemptsCacheEntry struct {
	CreatedAt      time.Time
	ExpiresAt      time.Time
	FallbackUsed   bool
	AttemptsByTarg map[int64][]segmentAttemptRaw
}

type segmentAnalysisCacheFile struct {
	SchemaVersion int                             `json:"schemaVersion"`
	GeneratedAt   string                          `json:"generatedAt"`
	Entries       []segmentAnalysisCacheDiskEntry `json:"entries"`
}

type segmentAnalysisCacheDiskEntry struct {
	Key          string                   `json:"key"`
	CreatedAt    string                   `json:"createdAt"`
	ExpiresAt    string                   `json:"expiresAt"`
	FallbackUsed bool                     `json:"fallbackUsed"`
	Attempts     []segmentAttemptRawCache `json:"attempts"`
}

type segmentAttemptRawCache struct {
	EffortID           int64                  `json:"effortId"`
	TargetID           int64                  `json:"targetId"`
	TargetName         string                 `json:"targetName"`
	TargetType         string                 `json:"targetType"`
	ClimbCategory      int                    `json:"climbCategory"`
	Distance           float64                `json:"distance"`
	AverageGrade       float64                `json:"averageGrade"`
	ElapsedTimeSeconds int                    `json:"elapsedTimeSeconds"`
	MovingTimeSeconds  int                    `json:"movingTimeSeconds"`
	SpeedKph           float64                `json:"speedKph"`
	AveragePowerWatts  float64                `json:"averagePowerWatts"`
	AverageHeartRate   float64                `json:"averageHeartRate"`
	ActivityDate       string                 `json:"activityDate"`
	PrRank             *int                   `json:"prRank,omitempty"`
	Activity           business.ActivityShort `json:"activity"`
}

func buildSegmentAttemptsCacheKey(
	year *int,
	from *string,
	to *string,
	activityTypes []business.ActivityType,
	activitySignature uint64,
) string {
	activityTypeNames := make([]string, 0, len(activityTypes))
	for _, activityType := range activityTypes {
		activityTypeNames = append(activityTypeNames, activityType.String())
	}
	sort.Strings(activityTypeNames)

	return strings.Join(
		[]string{
			fmt.Sprintf("algo:%s", segmentAnalysisAlgorithmVersion),
			fmt.Sprintf("types:%s", strings.Join(activityTypeNames, ",")),
			fmt.Sprintf("year:%s", normalizeOptionalInt(year)),
			fmt.Sprintf("from:%s", normalizeOptionalString(from)),
			fmt.Sprintf("to:%s", normalizeOptionalString(to)),
			fmt.Sprintf("activities:%x", activitySignature),
		},
		"|",
	)
}

func computeSegmentActivitiesSignature(activities []*strava.Activity) uint64 {
	hasher := fnv.New64a()
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		_, _ = hasher.Write([]byte(strconv.FormatInt(activity.Id, 10)))
		_, _ = hasher.Write([]byte("|"))
		_, _ = hasher.Write([]byte(activity.StartDateLocal))
		_, _ = hasher.Write([]byte("|"))
		_, _ = hasher.Write([]byte(activity.Name))
		_, _ = hasher.Write([]byte("|"))
		_, _ = hasher.Write([]byte(fmt.Sprintf("%.3f|%.3f|%d|%d|%s|%s", activity.Distance, activity.TotalElevationGain, activity.ElapsedTime, activity.MovingTime, activity.SportType, activity.Type)))
		_, _ = hasher.Write([]byte(";"))
	}
	return hasher.Sum64()
}

func getSegmentAttemptsFromCache(cacheKey string) (map[int64][]segmentAttemptRaw, bool) {
	ensureSegmentAttemptsCacheLoaded()

	now := time.Now().UTC()
	segmentAttemptsCache.RLock()
	entry, ok := segmentAttemptsCache.entries[cacheKey]
	segmentAttemptsCache.RUnlock()
	if !ok {
		return nil, false
	}

	if !entry.ExpiresAt.After(now) {
		segmentAttemptsCache.Lock()
		delete(segmentAttemptsCache.entries, cacheKey)
		persistSegmentAttemptsCacheLocked()
		segmentAttemptsCache.Unlock()
		return nil, false
	}

	return cloneAttemptsByTarget(entry.AttemptsByTarg), true
}

func storeSegmentAttemptsInCache(
	cacheKey string,
	attemptsByTarget map[int64][]segmentAttemptRaw,
	fallbackUsed bool,
) {
	ensureSegmentAttemptsCacheLoaded()

	if len(attemptsByTarget) == 0 {
		return
	}

	now := time.Now().UTC()
	ttl := segmentAnalysisCacheTTL
	if fallbackUsed {
		ttl = segmentAnalysisFallbackCacheTTL
	}

	entry := segmentAttemptsCacheEntry{
		CreatedAt:      now,
		ExpiresAt:      now.Add(ttl),
		FallbackUsed:   fallbackUsed,
		AttemptsByTarg: cloneAttemptsByTarget(attemptsByTarget),
	}

	segmentAttemptsCache.Lock()
	segmentAttemptsCache.entries[cacheKey] = entry
	trimSegmentAttemptsCacheLocked(now)
	persistSegmentAttemptsCacheLocked()
	segmentAttemptsCache.Unlock()
}

func ensureSegmentAttemptsCacheLoaded() {
	segmentAttemptsCache.Lock()
	defer segmentAttemptsCache.Unlock()
	if segmentAttemptsCache.loaded {
		return
	}

	segmentAttemptsCache.loaded = true
	cachePath, ok := segmentAnalysisCachePath()
	if !ok {
		return
	}

	payload, err := os.ReadFile(cachePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Unable to read segment analysis cache file: %v", err)
		}
		return
	}

	var disk segmentAnalysisCacheFile
	if err := json.Unmarshal(payload, &disk); err != nil {
		log.Printf("Unable to decode segment analysis cache file: %v", err)
		return
	}

	now := time.Now().UTC()
	loadedEntries := 0
	for _, diskEntry := range disk.Entries {
		if diskEntry.Key == "" {
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339, diskEntry.ExpiresAt)
		if err != nil || !expiresAt.After(now) {
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, diskEntry.CreatedAt)
		if err != nil {
			createdAt = now
		}
		attemptsByTarget := groupSegmentAttemptsByTarget(diskEntry.Attempts)
		if len(attemptsByTarget) == 0 {
			continue
		}
		segmentAttemptsCache.entries[diskEntry.Key] = segmentAttemptsCacheEntry{
			CreatedAt:      createdAt,
			ExpiresAt:      expiresAt,
			FallbackUsed:   diskEntry.FallbackUsed,
			AttemptsByTarg: attemptsByTarget,
		}
		loadedEntries++
	}

	if loadedEntries > 0 {
		log.Printf("Loaded segment analysis cache: %d entries", loadedEntries)
	}
}

func trimSegmentAttemptsCacheLocked(now time.Time) {
	for key, entry := range segmentAttemptsCache.entries {
		if !entry.ExpiresAt.After(now) {
			delete(segmentAttemptsCache.entries, key)
		}
	}

	if len(segmentAttemptsCache.entries) <= segmentAnalysisCacheMaxEntries {
		return
	}

	type sortableEntry struct {
		key       string
		createdAt time.Time
	}
	sortedEntries := make([]sortableEntry, 0, len(segmentAttemptsCache.entries))
	for key, entry := range segmentAttemptsCache.entries {
		sortedEntries = append(sortedEntries, sortableEntry{
			key:       key,
			createdAt: entry.CreatedAt,
		})
	}
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].createdAt.After(sortedEntries[j].createdAt)
	})

	for index := segmentAnalysisCacheMaxEntries; index < len(sortedEntries); index++ {
		delete(segmentAttemptsCache.entries, sortedEntries[index].key)
	}
}

func persistSegmentAttemptsCacheLocked() {
	cachePath, ok := segmentAnalysisCachePath()
	if !ok {
		return
	}

	now := time.Now().UTC()
	diskEntries := make([]segmentAnalysisCacheDiskEntry, 0, len(segmentAttemptsCache.entries))
	for key, entry := range segmentAttemptsCache.entries {
		if !entry.ExpiresAt.After(now) {
			continue
		}
		diskEntries = append(diskEntries, segmentAnalysisCacheDiskEntry{
			Key:          key,
			CreatedAt:    entry.CreatedAt.Format(time.RFC3339),
			ExpiresAt:    entry.ExpiresAt.Format(time.RFC3339),
			FallbackUsed: entry.FallbackUsed,
			Attempts:     flattenSegmentAttempts(entry.AttemptsByTarg),
		})
	}

	sort.Slice(diskEntries, func(i, j int) bool {
		return diskEntries[i].CreatedAt > diskEntries[j].CreatedAt
	})

	payload := segmentAnalysisCacheFile{
		SchemaVersion: segmentAnalysisCacheSchemaVersion,
		GeneratedAt:   now.Format(time.RFC3339),
		Entries:       diskEntries,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Printf("Unable to marshal segment analysis cache file: %v", err)
		return
	}

	if err := writeSegmentAnalysisCacheAtomically(cachePath, data); err != nil {
		log.Printf("Unable to save segment analysis cache file: %v", err)
	}
}

func writeSegmentAnalysisCacheAtomically(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func segmentAnalysisCachePath() (string, bool) {
	provider := getActivityProvider()
	cacheRoot := strings.TrimSpace(provider.CacheRootPath())
	clientID := strings.TrimSpace(provider.ClientID())
	if cacheRoot == "" || clientID == "" {
		return "", false
	}

	return filepath.Join(cacheRoot, fmt.Sprintf("strava-%s", clientID), segmentAnalysisCacheFileName), true
}

func groupSegmentAttemptsByTarget(flattened []segmentAttemptRawCache) map[int64][]segmentAttemptRaw {
	grouped := make(map[int64][]segmentAttemptRaw)
	for _, attempt := range flattened {
		targetType := segmentTargetTypeSegment
		if strings.EqualFold(attempt.TargetType, string(segmentTargetTypeClimb)) {
			targetType = segmentTargetTypeClimb
		}

		grouped[attempt.TargetID] = append(grouped[attempt.TargetID], segmentAttemptRaw{
			effortId:           attempt.EffortID,
			targetId:           attempt.TargetID,
			targetName:         attempt.TargetName,
			targetType:         targetType,
			direction:          segmentDirectionUnknown,
			climbCategory:      attempt.ClimbCategory,
			distance:           attempt.Distance,
			averageGrade:       attempt.AverageGrade,
			elapsedTimeSeconds: attempt.ElapsedTimeSeconds,
			movingTimeSeconds:  attempt.MovingTimeSeconds,
			speedKph:           attempt.SpeedKph,
			averagePowerWatts:  attempt.AveragePowerWatts,
			averageHeartRate:   attempt.AverageHeartRate,
			activityDate:       attempt.ActivityDate,
			prRank:             attempt.PrRank,
			activity:           attempt.Activity,
		})
	}
	return grouped
}

func flattenSegmentAttempts(attemptsByTarget map[int64][]segmentAttemptRaw) []segmentAttemptRawCache {
	flattened := make([]segmentAttemptRawCache, 0)
	for _, attempts := range attemptsByTarget {
		for _, attempt := range attempts {
			flattened = append(flattened, segmentAttemptRawCache{
				EffortID:           attempt.effortId,
				TargetID:           attempt.targetId,
				TargetName:         attempt.targetName,
				TargetType:         string(attempt.targetType),
				ClimbCategory:      attempt.climbCategory,
				Distance:           attempt.distance,
				AverageGrade:       attempt.averageGrade,
				ElapsedTimeSeconds: attempt.elapsedTimeSeconds,
				MovingTimeSeconds:  attempt.movingTimeSeconds,
				SpeedKph:           attempt.speedKph,
				AveragePowerWatts:  attempt.averagePowerWatts,
				AverageHeartRate:   attempt.averageHeartRate,
				ActivityDate:       attempt.activityDate,
				PrRank:             attempt.prRank,
				Activity:           attempt.activity,
			})
		}
	}
	return flattened
}

func cloneAttemptsByTarget(source map[int64][]segmentAttemptRaw) map[int64][]segmentAttemptRaw {
	cloned := make(map[int64][]segmentAttemptRaw, len(source))
	for targetID, attempts := range source {
		cloned[targetID] = append([]segmentAttemptRaw(nil), attempts...)
	}
	return cloned
}

func normalizeOptionalString(value *string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "none"
	}
	return strings.TrimSpace(*value)
}

func normalizeOptionalInt(value *int) string {
	if value == nil {
		return "all"
	}
	return strconv.Itoa(*value)
}
