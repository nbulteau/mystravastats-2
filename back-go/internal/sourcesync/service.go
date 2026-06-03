package sourcesync

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mystravastats/internal/helpers"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/platform/runtimeconfig"
	"mystravastats/internal/shared/domain/strava"
	fitprovider "mystravastats/internal/shared/infrastructure/fit"
)

const (
	fitDestinationEnv  = "FIT_FILES_PATH"
	garminSourceEnv    = "GARMIN_FIT_SOURCE_PATH"
	defaultVolumesRoot = "/Volumes"
)

type SyncResult struct {
	Status      string          `json:"status"`
	Reason      string          `json:"reason"`
	Message     string          `json:"message"`
	StartedAt   string          `json:"startedAt"`
	CompletedAt string          `json:"completedAt"`
	DurationMs  int64           `json:"durationMs"`
	Reloaded    bool            `json:"reloaded"`
	FIT         FITImportResult `json:"fit"`
}

type FITImportResult struct {
	Status                 string            `json:"status"`
	Message                string            `json:"message"`
	Configured             bool              `json:"configured"`
	SourcePath             string            `json:"sourcePath"`
	CandidateSourcePaths   []string          `json:"candidateSourcePaths"`
	DestinationPath        string            `json:"destinationPath"`
	ScannedFiles           int               `json:"scannedFiles"`
	ImportedFiles          int               `json:"importedFiles"`
	AlreadyPresentFiles    int               `json:"alreadyPresentFiles"`
	SkippedFiles           int               `json:"skippedFiles"`
	InvalidFiles           int               `json:"invalidFiles"`
	CreatedYearDirectories []string          `json:"createdYearDirectories"`
	Imported               []ImportedFITFile `json:"imported"`
	Errors                 []string          `json:"errors"`
}

type ImportedFITFile struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Year        string `json:"year"`
	ActivityID  int64  `json:"activityId"`
	StartDate   string `json:"startDate"`
}

type Service struct {
	decodeFIT        func(filePath string, athleteID int64) (*strava.Activity, error)
	fitDestination   func() (string, bool)
	garminSourcePath func() (string, bool)
	reloadProvider   func()
	volumesRoot      string
	now              func() time.Time
	running          atomic.Bool
	lastResultMutex  sync.RWMutex
	lastResult       SyncResult
}

var defaultService = NewService()

func NewService() *Service {
	return &Service{
		decodeFIT:        fitprovider.DecodeFITActivity,
		fitDestination:   func() (string, bool) { return runtimeconfig.OptionalValue(fitDestinationEnv) },
		garminSourcePath: func() (string, bool) { return runtimeconfig.OptionalValue(garminSourceEnv) },
		reloadProvider:   activityprovider.Reload,
		volumesRoot:      defaultVolumesRoot,
		now:              time.Now,
		lastResult: SyncResult{
			Status:  "idle",
			Message: "Synchronization has not run yet.",
			FIT: FITImportResult{
				Status:  "idle",
				Message: "FIT import has not run yet.",
			},
		},
	}
}

func Synchronize(reason string) SyncResult {
	return defaultService.Synchronize(reason)
}

func LastResult() SyncResult {
	return defaultService.LastResult()
}

func (service *Service) Synchronize(reason string) SyncResult {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "manual"
	}
	if !service.running.CompareAndSwap(false, true) {
		result := service.LastResult()
		result.Status = "running"
		result.Reason = reason
		result.Message = "Synchronization is already running."
		return result
	}
	defer service.running.Store(false)

	startedAt := service.now()
	result := SyncResult{
		Status:    "completed",
		Reason:    reason,
		StartedAt: startedAt.UTC().Format(time.RFC3339),
	}
	result.FIT = service.importFIT()
	if result.FIT.ImportedFiles > 0 {
		if service.reloadProvider != nil {
			service.reloadProvider()
			result.Reloaded = true
		}
	}
	result.Status = syncStatusFromFIT(result.FIT)
	result.Message = syncMessageFromFIT(result.FIT)
	completedAt := service.now()
	result.CompletedAt = completedAt.UTC().Format(time.RFC3339)
	result.DurationMs = completedAt.Sub(startedAt).Milliseconds()
	service.storeLastResult(result)
	return result
}

func (service *Service) LastResult() SyncResult {
	service.lastResultMutex.RLock()
	defer service.lastResultMutex.RUnlock()
	return service.lastResult
}

func (service *Service) storeLastResult(result SyncResult) {
	service.lastResultMutex.Lock()
	service.lastResult = result
	service.lastResultMutex.Unlock()
}

func (service *Service) importFIT() FITImportResult {
	destinationPath, configured := service.fitDestination()
	result := FITImportResult{
		Status:          "not_configured",
		Configured:      configured,
		DestinationPath: strings.TrimSpace(destinationPath),
	}
	if strings.TrimSpace(destinationPath) == "" {
		result.Message = "FIT directory is not configured."
		return result
	}

	sourcePath, candidates := service.detectGarminFITSource()
	result.CandidateSourcePaths = candidates
	if sourcePath == "" {
		result.Status = "no_device"
		result.Message = "No Garmin USB activity directory was detected."
		return result
	}
	result.SourcePath = sourcePath

	if err := os.MkdirAll(destinationPath, 0o755); err != nil {
		result.Status = "failed"
		result.Message = "Unable to create FIT destination directory."
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	existingFingerprints := service.existingFITFingerprints(destinationPath, &result)
	createdYears := map[string]struct{}{}

	err := filepath.WalkDir(sourcePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.Errors = append(result.Errors, walkErr.Error())
			return nil
		}
		if entry == nil || entry.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".fit") {
			return nil
		}
		result.ScannedFiles++
		activity, decodeErr := service.decodeFIT(path, 0)
		if decodeErr != nil {
			result.InvalidFiles++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, decodeErr))
			return nil
		}
		fingerprint := activityFingerprint(activity)
		if fingerprint != "" && existingFingerprints[fingerprint] {
			result.AlreadyPresentFiles++
			result.SkippedFiles++
			return nil
		}

		year := activityYear(activity)
		yearDirectory := filepath.Join(destinationPath, year)
		if _, statErr := os.Stat(yearDirectory); errors.Is(statErr, os.ErrNotExist) {
			createdYears[year] = struct{}{}
		}
		if mkdirErr := os.MkdirAll(yearDirectory, 0o755); mkdirErr != nil {
			result.InvalidFiles++
			result.Errors = append(result.Errors, mkdirErr.Error())
			return nil
		}

		destinationFile, nameErr := service.destinationFilePath(path, yearDirectory)
		if nameErr != nil {
			result.InvalidFiles++
			result.Errors = append(result.Errors, nameErr.Error())
			return nil
		}
		if copyErr := copyFileAtomic(path, destinationFile); copyErr != nil {
			result.InvalidFiles++
			result.Errors = append(result.Errors, copyErr.Error())
			return nil
		}

		result.ImportedFiles++
		if fingerprint != "" {
			existingFingerprints[fingerprint] = true
		}
		if len(result.Imported) < 25 {
			result.Imported = append(result.Imported, ImportedFITFile{
				Source:      path,
				Destination: destinationFile,
				Year:        year,
				ActivityID:  activity.Id,
				StartDate:   helpers.FirstNonEmpty(activity.StartDateLocal, activity.StartDate),
			})
		}
		return nil
	})
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	result.CreatedYearDirectories = sortedMapKeys(createdYears)
	result.Status = fitImportStatus(result)
	result.Message = fitImportMessage(result)
	if len(result.Errors) > 0 {
		log.Printf("FIT import completed with %d error(s)", len(result.Errors))
	}
	return result
}

func (service *Service) detectGarminFITSource() (string, []string) {
	candidates := make([]string, 0)
	if configuredPath, configured := service.garminSourcePath(); configured && strings.TrimSpace(configuredPath) != "" {
		path := filepath.Clean(strings.TrimSpace(configuredPath))
		candidates = append(candidates, path)
		if isDirectory(path) {
			return path, candidates
		}
	}

	volumeEntries, err := os.ReadDir(service.volumesRoot)
	if err != nil {
		return "", candidates
	}
	for _, volumeEntry := range volumeEntries {
		if !volumeEntry.IsDir() {
			continue
		}
		if strings.HasPrefix(volumeEntry.Name(), ".") {
			continue
		}
		volumeRoot := filepath.Join(service.volumesRoot, volumeEntry.Name())
		for _, parts := range [][]string{
			{"GARMIN", "ACTIVITY"},
			{"GARMIN", "Activity"},
			{"Garmin", "ACTIVITY"},
			{"Garmin", "Activity"},
		} {
			candidate := filepath.Join(append([]string{volumeRoot}, parts...)...)
			candidates = append(candidates, candidate)
			if isDirectory(candidate) {
				return candidate, candidates
			}
		}
	}
	return "", candidates
}

func (service *Service) existingFITFingerprints(destinationPath string, result *FITImportResult) map[string]bool {
	fingerprints := map[string]bool{}
	_ = filepath.WalkDir(destinationPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".fit") {
			return nil
		}
		activity, err := service.decodeFIT(path, 0)
		if err != nil {
			return nil
		}
		if fingerprint := activityFingerprint(activity); fingerprint != "" {
			fingerprints[fingerprint] = true
		}
		return nil
	})
	return fingerprints
}

func (service *Service) destinationFilePath(sourcePath string, yearDirectory string) (string, error) {
	base := filepath.Base(sourcePath)
	if strings.TrimSpace(base) == "" || base == "." || base == string(filepath.Separator) {
		base = "activity.fit"
	}
	destination := filepath.Join(yearDirectory, base)
	if _, err := os.Stat(destination); errors.Is(err, os.ErrNotExist) {
		return destination, nil
	}
	hash, err := shortFileHash(sourcePath)
	if err != nil {
		return "", err
	}
	extension := filepath.Ext(base)
	stem := strings.TrimSuffix(base, extension)
	if stem == "" {
		stem = "activity"
	}
	for index := 0; index < 100; index++ {
		suffix := hash
		if index > 0 {
			suffix = fmt.Sprintf("%s-%d", hash, index)
		}
		candidate := filepath.Join(yearDirectory, fmt.Sprintf("%s-%s%s", stem, suffix, extension))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("unable to choose destination file for %s", sourcePath)
}

func activityFingerprint(activity *strava.Activity) string {
	if activity == nil {
		return ""
	}
	start := helpers.FirstNonEmpty(activity.StartDate, activity.StartDateLocal)
	parsedStart, ok := helpers.ParseActivityDate(start)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s|%s|%.0f|%d",
		strings.TrimSpace(activity.Type),
		parsedStart.UTC().Format(time.RFC3339),
		math.Round(activity.Distance),
		activity.ElapsedTime,
	)
}

func activityYear(activity *strava.Activity) string {
	if activity != nil {
		for _, value := range []string{activity.StartDateLocal, activity.StartDate} {
			if parsed, ok := helpers.ParseActivityDate(value); ok {
				return strconv.Itoa(parsed.Year())
			}
			if len(strings.TrimSpace(value)) >= 4 {
				candidate := strings.TrimSpace(value)[:4]
				if _, err := strconv.Atoi(candidate); err == nil {
					return candidate
				}
			}
		}
	}
	return strconv.Itoa(time.Now().Year())
}

func copyFileAtomic(sourcePath string, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	tempPath := destinationPath + ".tmp"
	destination, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return closeErr
	}
	return os.Rename(tempPath, destinationPath)
}

func shortFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil))[:10], nil
}

func syncStatusFromFIT(fit FITImportResult) string {
	switch fit.Status {
	case "failed":
		return "failed"
	case "not_configured", "no_device":
		return "skipped"
	default:
		return "completed"
	}
}

func syncMessageFromFIT(fit FITImportResult) string {
	switch fit.Status {
	case "imported":
		return fmt.Sprintf("Imported %d FIT file(s).", fit.ImportedFiles)
	case "up_to_date":
		return "FIT library is already up to date."
	case "no_files":
		return "Garmin USB source was found, but no FIT files were present."
	case "no_device":
		return "No Garmin USB activity directory was detected."
	case "not_configured":
		return "FIT import skipped because FIT_FILES_PATH is not configured."
	case "failed":
		return "FIT import failed."
	default:
		return fit.Message
	}
}

func fitImportStatus(result FITImportResult) string {
	if result.ScannedFiles == 0 {
		return "no_files"
	}
	if result.ImportedFiles > 0 {
		return "imported"
	}
	if result.InvalidFiles > 0 && result.InvalidFiles == result.ScannedFiles {
		return "failed"
	}
	return "up_to_date"
}

func fitImportMessage(result FITImportResult) string {
	switch result.Status {
	case "imported":
		return fmt.Sprintf("%d new FIT file(s) imported into year folders.", result.ImportedFiles)
	case "up_to_date":
		return fmt.Sprintf("%d FIT file(s) already present.", result.AlreadyPresentFiles)
	case "no_files":
		return "No FIT files found in the detected Garmin activity directory."
	case "failed":
		return "No valid FIT file could be imported."
	default:
		return result.Message
	}
}

func sortedMapKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
