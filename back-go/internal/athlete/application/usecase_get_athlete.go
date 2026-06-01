package application

import (
	"sort"
	"time"

	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

type GetAthleteUseCase struct {
	reader AthleteReader
}

func NewGetAthleteUseCase(reader AthleteReader) *GetAthleteUseCase {
	return &GetAthleteUseCase{
		reader: reader,
	}
}

func (uc *GetAthleteUseCase) Execute() strava.Athlete {
	return uc.reader.FindAthlete()
}

type GetPerformanceSettingsUseCase struct {
	reader AthleteReader
}

func NewGetPerformanceSettingsUseCase(reader AthleteReader) *GetPerformanceSettingsUseCase {
	return &GetPerformanceSettingsUseCase{reader: reader}
}

func (uc *GetPerformanceSettingsUseCase) Execute() business.AthletePerformanceSettings {
	return normalizePerformanceSettings(uc.reader.FindPerformanceSettings())
}

type UpdatePerformanceSettingsUseCase struct {
	reader AthleteReader
}

func NewUpdatePerformanceSettingsUseCase(reader AthleteReader) *UpdatePerformanceSettingsUseCase {
	return &UpdatePerformanceSettingsUseCase{reader: reader}
}

func (uc *UpdatePerformanceSettingsUseCase) Execute(settings business.AthletePerformanceSettings) business.AthletePerformanceSettings {
	return uc.reader.SavePerformanceSettings(normalizePerformanceSettings(settings))
}

func normalizePerformanceSettings(settings business.AthletePerformanceSettings) business.AthletePerformanceSettings {
	normalized := business.AthletePerformanceSettings{}
	if settings.WeightKg != nil && *settings.WeightKg > 0 {
		weight := *settings.WeightKg
		normalized.WeightKg = &weight
	}

	byDate := make(map[string]business.AthleteFtpSetting)
	for _, entry := range settings.FtpHistory {
		if entry.Ftp <= 0 {
			continue
		}
		if _, err := time.Parse("2006-01-02", entry.EffectiveFrom); err != nil {
			continue
		}
		byDate[entry.EffectiveFrom] = business.AthleteFtpSetting{
			EffectiveFrom: entry.EffectiveFrom,
			Ftp:           entry.Ftp,
		}
	}

	for _, entry := range byDate {
		normalized.FtpHistory = append(normalized.FtpHistory, entry)
	}
	sort.Slice(normalized.FtpHistory, func(i, j int) bool {
		return normalized.FtpHistory[i].EffectiveFrom < normalized.FtpHistory[j].EffectiveFrom
	})

	return normalized
}
