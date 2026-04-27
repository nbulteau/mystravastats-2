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
	gearMaintenanceSecureDirMode  = 0700
	gearMaintenanceSecureFileMode = 0600
)

type gearMaintenanceFile struct {
	Records []business.GearMaintenanceRecord `json:"records"`
}

func loadCurrentProviderGearMaintenanceRecords() []business.GearMaintenanceRecord {
	provider := activityprovider.Get()
	return loadGearMaintenanceRecords(provider.CacheRootPath(), provider.ClientID())
}

func saveCurrentProviderGearMaintenanceRecord(request business.GearMaintenanceRecordRequest) (business.GearMaintenanceRecord, error) {
	provider := activityprovider.Get()
	normalized, err := normalizeGearMaintenanceRequest(request)
	if err != nil {
		return business.GearMaintenanceRecord{}, err
	}

	athlete := provider.GetAthlete()
	gearName := gearNameForMaintenance(athlete, normalized.GearID)
	if gearName == "" {
		gearName = gearDisplayName(normalized.GearID, gearMetadata{kind: inferGearKind(normalized.GearID)})
	}

	records := loadGearMaintenanceRecords(provider.CacheRootPath(), provider.ClientID())
	now := time.Now().UTC().Format(time.RFC3339)
	record := business.GearMaintenanceRecord{
		ID:             fmt.Sprintf("gm-%d", time.Now().UTC().UnixNano()),
		GearID:         normalized.GearID,
		GearName:       gearName,
		Component:      normalized.Component,
		ComponentLabel: gearMaintenanceComponentLabel(normalized.Component),
		Operation:      normalized.Operation,
		Date:           normalized.Date,
		Distance:       roundGearValue(normalized.Distance),
		Note:           normalized.Note,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	records = append(records, record)
	if err := saveGearMaintenanceRecords(provider.CacheRootPath(), provider.ClientID(), records); err != nil {
		return business.GearMaintenanceRecord{}, err
	}
	return record, nil
}

func deleteCurrentProviderGearMaintenanceRecord(recordID string) error {
	provider := activityprovider.Get()
	trimmedID := strings.TrimSpace(recordID)
	if trimmedID == "" {
		return fmt.Errorf("recordId is required")
	}

	records := loadGearMaintenanceRecords(provider.CacheRootPath(), provider.ClientID())
	updated := make([]business.GearMaintenanceRecord, 0, len(records))
	found := false
	for _, record := range records {
		if record.ID == trimmedID {
			found = true
			continue
		}
		updated = append(updated, record)
	}
	if !found {
		return fmt.Errorf("maintenance record %s not found", trimmedID)
	}
	return saveGearMaintenanceRecords(provider.CacheRootPath(), provider.ClientID(), updated)
}

func loadGearMaintenanceRecords(cacheRoot string, clientID string) []business.GearMaintenanceRecord {
	path := gearMaintenanceFilePath(cacheRoot, clientID)
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to read gear maintenance file '%s': %v", path, err)
		}
		return []business.GearMaintenanceRecord{}
	}

	payload := gearMaintenanceFile{}
	if err := json.Unmarshal(data, &payload); err != nil {
		log.Printf("Failed to unmarshal gear maintenance file '%s': %v", path, err)
		return []business.GearMaintenanceRecord{}
	}
	return normalizeGearMaintenanceRecords(payload.Records)
}

func saveGearMaintenanceRecords(cacheRoot string, clientID string, records []business.GearMaintenanceRecord) error {
	athleteDirectory := gearMaintenanceDirectory(cacheRoot, clientID)
	if err := os.MkdirAll(athleteDirectory, gearMaintenanceSecureDirMode); err != nil {
		return fmt.Errorf("unable to create gear maintenance directory: %w", err)
	}

	normalized := normalizeGearMaintenanceRecords(records)
	payload := gearMaintenanceFile{Records: normalized}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to encode gear maintenance records: %w", err)
	}
	if err := os.WriteFile(gearMaintenanceFilePath(cacheRoot, clientID), data, gearMaintenanceSecureFileMode); err != nil {
		return fmt.Errorf("unable to write gear maintenance records: %w", err)
	}
	return nil
}

func normalizeGearMaintenanceRequest(request business.GearMaintenanceRecordRequest) (business.GearMaintenanceRecordRequest, error) {
	normalized := business.GearMaintenanceRecordRequest{
		GearID:    strings.TrimSpace(request.GearID),
		Component: normalizeGearMaintenanceComponent(request.Component),
		Operation: strings.TrimSpace(request.Operation),
		Date:      strings.TrimSpace(request.Date),
		Distance:  request.Distance,
		Note:      strings.TrimSpace(request.Note),
	}
	if normalized.GearID == "" {
		return normalized, fmt.Errorf("gearId is required")
	}
	if normalized.Component == "" {
		return normalized, fmt.Errorf("component is required")
	}
	if normalized.Operation == "" {
		normalized.Operation = fmt.Sprintf("%s serviced", gearMaintenanceComponentLabel(normalized.Component))
	}
	if len(normalized.Date) >= 10 {
		normalized.Date = normalized.Date[:10]
	}
	if _, err := time.Parse("2006-01-02", normalized.Date); err != nil {
		return normalized, fmt.Errorf("date must use YYYY-MM-DD")
	}
	if normalized.Distance < 0 {
		return normalized, fmt.Errorf("distance must be >= 0")
	}
	return normalized, nil
}

func normalizeGearMaintenanceRecords(records []business.GearMaintenanceRecord) []business.GearMaintenanceRecord {
	normalized := make([]business.GearMaintenanceRecord, 0, len(records))
	for _, record := range records {
		record.ID = strings.TrimSpace(record.ID)
		record.GearID = strings.TrimSpace(record.GearID)
		record.Component = normalizeGearMaintenanceComponent(record.Component)
		if record.ID == "" || record.GearID == "" || record.Component == "" {
			continue
		}
		record.GearName = strings.TrimSpace(record.GearName)
		record.ComponentLabel = gearMaintenanceComponentLabel(record.Component)
		record.Operation = strings.TrimSpace(record.Operation)
		record.Date = strings.TrimSpace(record.Date)
		if len(record.Date) >= 10 {
			record.Date = record.Date[:10]
		}
		record.Note = strings.TrimSpace(record.Note)
		record.Distance = roundGearValue(record.Distance)
		normalized = append(normalized, record)
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].GearID == normalized[j].GearID {
			if normalized[i].Date == normalized[j].Date {
				return normalized[i].CreatedAt > normalized[j].CreatedAt
			}
			return normalized[i].Date > normalized[j].Date
		}
		return normalized[i].GearID < normalized[j].GearID
	})
	return normalized
}

func normalizeGearMaintenanceComponent(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	for _, rule := range bikeMaintenanceRules {
		if rule.component == normalized {
			return normalized
		}
	}
	return ""
}

func gearMaintenanceComponentLabel(component string) string {
	for _, rule := range bikeMaintenanceRules {
		if rule.component == component {
			return rule.label
		}
	}
	return component
}

func gearNameForMaintenance(athlete strava.Athlete, gearID string) string {
	for _, bike := range athlete.Bikes {
		if strings.TrimSpace(bike.Id) == gearID {
			return firstNonBlankString(bike.Nickname, bike.Name)
		}
	}
	return ""
}

func gearMaintenanceDirectory(cacheRoot string, clientID string) string {
	return filepath.Join(cacheRoot, fmt.Sprintf("strava-%s", clientID))
}

func gearMaintenanceFilePath(cacheRoot string, clientID string) string {
	return filepath.Join(gearMaintenanceDirectory(cacheRoot, clientID), fmt.Sprintf("gear-maintenance-%s.json", clientID))
}
