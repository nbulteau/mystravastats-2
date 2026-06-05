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
	"runtime"
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
	fitInboxEnv        = "FIT_INBOX_PATH"
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
	Status                 string               `json:"status"`
	Message                string               `json:"message"`
	Configured             bool                 `json:"configured"`
	SourceKind             string               `json:"sourceKind"`
	SourcePath             string               `json:"sourcePath"`
	InboxPath              string               `json:"inboxPath"`
	CandidateSourcePaths   []string             `json:"candidateSourcePaths"`
	DestinationPath        string               `json:"destinationPath"`
	ScannedFiles           int                  `json:"scannedFiles"`
	ImportedFiles          int                  `json:"importedFiles"`
	AlreadyPresentFiles    int                  `json:"alreadyPresentFiles"`
	SkippedFiles           int                  `json:"skippedFiles"`
	InvalidFiles           int                  `json:"invalidFiles"`
	CreatedYearDirectories []string             `json:"createdYearDirectories"`
	Imported               []ImportedFITFile    `json:"imported"`
	Errors                 []string             `json:"errors"`
	DeviceSync             *FITDeviceSyncResult `json:"deviceSync,omitempty"`
}

type ImportedFITFile struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Year        string `json:"year"`
	ActivityID  int64  `json:"activityId"`
	StartDate   string `json:"startDate"`
}

type FITDeviceSyncResult struct {
	Status               string              `json:"status"`
	Message              string              `json:"message"`
	Backend              string              `json:"backend"`
	Device               string              `json:"device,omitempty"`
	SourcePath           string              `json:"sourcePath,omitempty"`
	InboxPath            string              `json:"inboxPath,omitempty"`
	CandidateSourcePaths []string            `json:"candidateSourcePaths,omitempty"`
	ScannedFiles         int                 `json:"scannedFiles"`
	CopiedFiles          int                 `json:"copiedFiles"`
	AlreadyPresentFiles  int                 `json:"alreadyPresentFiles"`
	SkippedFiles         int                 `json:"skippedFiles"`
	InvalidFiles         int                 `json:"invalidFiles"`
	Copied               []FITDeviceSyncFile `json:"copied,omitempty"`
	Errors               []string            `json:"errors,omitempty"`
}

type FITDeviceSyncFile struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type Service struct {
	decodeFIT        func(filePath string, athleteID int64) (*strava.Activity, error)
	fitDestination   func() (string, bool)
	fitInboxPath     func() (string, bool)
	garminSourcePath func() (string, bool)
	reloadProvider   func()
	volumeRoots      func() []string
	now              func() time.Time
	running          atomic.Bool
	lastResultMutex  sync.RWMutex
	lastResult       SyncResult
}

type garminDevice struct {
	name         string
	root         string
	activityPath string
}

var defaultService = NewService()

func NewService() *Service {
	return &Service{
		decodeFIT:        fitprovider.DecodeFITActivity,
		fitDestination:   func() (string, bool) { return runtimeconfig.OptionalValue(fitDestinationEnv) },
		fitInboxPath:     configuredFITInboxPath,
		garminSourcePath: func() (string, bool) { return runtimeconfig.OptionalValue(garminSourceEnv) },
		reloadProvider:   activityprovider.Reload,
		volumeRoots:      platformVolumeRoots,
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

func configuredFITInboxPath() (string, bool) {
	value, configured, _ := runtimeconfig.FITInboxPath()
	return value, configured
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
	inboxPath, inboxConfigured := service.fitInboxPath()
	result := FITImportResult{
		Status:          "not_configured",
		Configured:      configured,
		DestinationPath: strings.TrimSpace(destinationPath),
		InboxPath:       strings.TrimSpace(inboxPath),
	}
	if strings.TrimSpace(destinationPath) == "" {
		result.Message = "FIT directory is not configured."
		return result
	}

	if strings.TrimSpace(inboxPath) != "" {
		if err := os.MkdirAll(strings.TrimSpace(inboxPath), 0o755); err != nil {
			result.Status = "failed"
			result.Message = "Unable to create FIT inbox directory."
			result.Errors = append(result.Errors, err.Error())
			return result
		}
	}

	if deviceSync := service.syncGarminToInbox(strings.TrimSpace(inboxPath)); deviceSync != nil {
		result.DeviceSync = deviceSync
		result.CandidateSourcePaths = append(result.CandidateSourcePaths, deviceSync.CandidateSourcePaths...)
		if deviceSync.Status == "failed" {
			result.Errors = append(result.Errors, deviceSync.Errors...)
		}
	}

	sourcePath, sourceKind, candidates := service.detectFITImportSource(strings.TrimSpace(inboxPath), inboxConfigured)
	result.CandidateSourcePaths = append(result.CandidateSourcePaths, candidates...)
	if sourcePath == "" {
		result.Status = "no_device"
		result.Message = "No FIT inbox or Garmin USB activity directory was detected."
		return result
	}
	result.SourceKind = sourceKind
	result.SourcePath = sourcePath

	if err := os.MkdirAll(destinationPath, 0o755); err != nil {
		result.Status = "failed"
		result.Message = "Unable to create FIT destination directory."
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	existingFingerprints := service.existingFITFingerprints(destinationPath, sourcePath, &result)
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

func (service *Service) syncGarminToInbox(inboxPath string) *FITDeviceSyncResult {
	if strings.TrimSpace(inboxPath) == "" {
		return nil
	}
	device, candidates := service.detectGarminDevice()
	result := FITDeviceSyncResult{
		Status:               "no_device",
		Message:              "No mounted Garmin activity directory was detected.",
		Backend:              "filesystem",
		InboxPath:            inboxPath,
		CandidateSourcePaths: candidates,
	}
	if device == nil {
		return &result
	}
	if err := os.MkdirAll(inboxPath, 0o755); err != nil {
		result.Status = "failed"
		result.Message = "Unable to create FIT inbox directory."
		result.Errors = append(result.Errors, err.Error())
		return &result
	}

	result.Status = "ok"
	result.Device = device.name
	result.SourcePath = device.activityPath
	files := fitFiles(device.activityPath)
	result.ScannedFiles = len(files)
	for _, file := range files {
		destination, copied, err := copyFITToInbox(file, inboxPath)
		if err != nil {
			result.InvalidFiles++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", file, err))
			continue
		}
		if copied {
			result.CopiedFiles++
			if len(result.Copied) < 25 {
				result.Copied = append(result.Copied, FITDeviceSyncFile{
					Source:      file,
					Destination: destination,
				})
			}
		} else {
			result.AlreadyPresentFiles++
			result.SkippedFiles++
		}
	}
	result.Message = garminDeviceSyncMessage(result)
	return &result
}

func (service *Service) detectFITImportSource(inboxPath string, inboxConfigured bool) (string, string, []string) {
	candidates := make([]string, 0)
	if inboxConfigured && inboxPath != "" {
		path := filepath.Clean(inboxPath)
		candidates = append(candidates, path)
		if isDirectory(path) {
			return path, "fit_inbox", candidates
		}
	}
	sourcePath, garminCandidates := service.detectGarminFITSource()
	candidates = append(candidates, garminCandidates...)
	if sourcePath != "" {
		return sourcePath, "garmin_usb", candidates
	}
	return "", "", candidates
}

func (service *Service) detectGarminFITSource() (string, []string) {
	device, candidates := service.detectGarminDevice()
	if device != nil {
		return device.activityPath, candidates
	}
	return "", candidates
}

func (service *Service) detectGarminDevice() (*garminDevice, []string) {
	candidates := make([]string, 0)
	if configuredPath, configured := service.garminSourcePath(); configured && strings.TrimSpace(configuredPath) != "" {
		appendGarminSourceCandidates(filepath.Clean(strings.TrimSpace(configuredPath)), &candidates)
	} else {
		for _, root := range service.volumeRoots() {
			appendGarminVolumeCandidates(root, &candidates)
		}
	}
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if isDirectory(candidate) {
			return &garminDevice{
				name:         garminDeviceName(candidate),
				root:         garminDeviceRoot(candidate),
				activityPath: candidate,
			}, candidates
		}
	}
	return nil, candidates
}

func (service *Service) existingFITFingerprints(destinationPath string, sourcePath string, result *FITImportResult) map[string]bool {
	fingerprints := map[string]bool{}
	cleanSourcePath := filepath.Clean(strings.TrimSpace(sourcePath))
	_ = filepath.WalkDir(destinationPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".fit") {
			return nil
		}
		if cleanSourcePath != "" && sameOrInside(path, cleanSourcePath) {
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

func copyFITToInbox(sourcePath string, inboxPath string) (string, bool, error) {
	base := filepath.Base(sourcePath)
	if strings.TrimSpace(base) == "" || base == "." || base == string(filepath.Separator) {
		base = "activity.fit"
	}
	preferred := filepath.Join(inboxPath, base)
	if sameFileSize(sourcePath, preferred) {
		return preferred, false, nil
	}
	destination, err := availableInboxDestination(sourcePath, preferred)
	if err != nil {
		return "", false, err
	}
	if err := copyFileAtomic(sourcePath, destination); err != nil {
		return "", false, err
	}
	return destination, true, nil
}

func sameFileSize(sourcePath string, destinationPath string) bool {
	sourceInfo, sourceErr := os.Stat(sourcePath)
	destinationInfo, destinationErr := os.Stat(destinationPath)
	return sourceErr == nil && destinationErr == nil && !sourceInfo.IsDir() && !destinationInfo.IsDir() && sourceInfo.Size() == destinationInfo.Size()
}

func availableInboxDestination(sourcePath string, preferred string) (string, error) {
	if _, err := os.Stat(preferred); errors.Is(err, os.ErrNotExist) {
		return preferred, nil
	}
	sourceInfo, _ := os.Stat(sourcePath)
	sourceSize := int64(0)
	if sourceInfo != nil {
		sourceSize = sourceInfo.Size()
	}
	extension := filepath.Ext(preferred)
	stem := strings.TrimSuffix(filepath.Base(preferred), extension)
	if stem == "" {
		stem = "activity"
	}
	if extension == "" {
		extension = ".fit"
	}
	parent := filepath.Dir(preferred)
	for index := 1; index < 1000; index++ {
		candidate := filepath.Join(parent, fmt.Sprintf("%s-%d-%d%s", stem, sourceSize, index, extension))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("unable to choose destination file for %s", sourcePath)
}

func fitFiles(root string) []string {
	files := make([]string, 0)
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".fit") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files
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
		if fit.SourceKind == "fit_inbox" {
			return "FIT inbox is configured, but no FIT files were present."
		}
		return "Garmin USB source was found, but no FIT files were present."
	case "no_device":
		return "No FIT inbox or Garmin USB activity directory was detected."
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
		if result.SourceKind == "fit_inbox" {
			return "No FIT files found in the configured FIT inbox."
		}
		return "No FIT files found in the detected Garmin activity directory."
	case "failed":
		return "No valid FIT file could be imported."
	default:
		return result.Message
	}
}

func garminDeviceSyncMessage(result FITDeviceSyncResult) string {
	if result.Status == "failed" {
		return "Garmin FIT synchronization failed."
	}
	if result.CopiedFiles > 0 {
		return fmt.Sprintf("Copied %d FIT file(s) from Garmin source to inbox.", result.CopiedFiles)
	}
	if result.ScannedFiles == 0 {
		return "Mounted Garmin activity directory was found, but it contains no FIT files."
	}
	return fmt.Sprintf("%d FIT file(s) already present in inbox.", result.AlreadyPresentFiles)
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

func appendGarminSourceCandidates(source string, candidates *[]string) {
	appendGarminActivitySubdirectories(source, candidates)
	*candidates = append(*candidates, source)
}

func appendGarminVolumeCandidates(root string, candidates *[]string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		appendGarminActivitySubdirectories(filepath.Join(root, entry.Name()), candidates)
	}
}

func appendGarminActivitySubdirectories(root string, candidates *[]string) {
	for _, parts := range [][]string{
		{"GARMIN", "ACTIVITY"},
		{"GARMIN", "Activity"},
		{"Garmin", "ACTIVITY"},
		{"Garmin", "Activity"},
	} {
		*candidates = append(*candidates, filepath.Join(append([]string{root}, parts...)...))
	}
}

func garminDeviceName(activityPath string) string {
	root := garminDeviceRoot(activityPath)
	name := filepath.Base(root)
	if strings.TrimSpace(name) == "" || name == "." || name == string(filepath.Separator) {
		return "Garmin"
	}
	return name
}

func garminDeviceRoot(activityPath string) string {
	parent := filepath.Dir(activityPath)
	if parent == "." || parent == string(filepath.Separator) {
		return activityPath
	}
	root := filepath.Dir(parent)
	if root == "." || root == string(filepath.Separator) {
		return activityPath
	}
	return root
}

func platformVolumeRoots() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{defaultVolumesRoot}
	case "windows":
		roots := make([]string, 0, 26)
		for letter := 'A'; letter <= 'Z'; letter++ {
			roots = append(roots, fmt.Sprintf("%c:\\", letter))
		}
		return roots
	case "linux":
		roots := make([]string, 0)
		if user := strings.TrimSpace(os.Getenv("USER")); user != "" {
			roots = append(roots, filepath.Join("/run/media", user), filepath.Join("/media", user))
		}
		return append(roots, "/media", "/mnt")
	default:
		return []string{}
	}
}

func sameOrInside(path string, root string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(root)
	if cleanPath == cleanRoot {
		return true
	}
	relative, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return false
	}
	return relative != "." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != ".."
}
