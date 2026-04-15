package application

import (
	"mystravastats/domain/business"
	dashboardDomain "mystravastats/internal/dashboard/domain"
)

type GetDashboardDataUseCase struct {
	reader DashboardReader
}

func NewGetDashboardDataUseCase(reader DashboardReader) *GetDashboardDataUseCase {
	return &GetDashboardDataUseCase{reader: reader}
}

func (uc *GetDashboardDataUseCase) Execute(activityTypes []business.ActivityType) business.DashboardData {
	return uc.reader.FindDashboardData(activityTypes...)
}

type GetCumulativeDataPerYearUseCase struct {
	reader DashboardReader
}

func NewGetCumulativeDataPerYearUseCase(reader DashboardReader) *GetCumulativeDataPerYearUseCase {
	return &GetCumulativeDataPerYearUseCase{reader: reader}
}

func (uc *GetCumulativeDataPerYearUseCase) Execute(activityTypes []business.ActivityType) dashboardDomain.CumulativeDataPerYear {
	distance := uc.reader.FindCumulativeDistancePerYear(activityTypes...)
	elevation := uc.reader.FindCumulativeElevationPerYear(activityTypes...)
	if distance == nil {
		distance = map[string]map[string]float64{}
	}
	if elevation == nil {
		elevation = map[string]map[string]float64{}
	}

	return dashboardDomain.CumulativeDataPerYear{
		Distance:  distance,
		Elevation: elevation,
	}
}

type GetActivityHeatmapUseCase struct {
	reader DashboardReader
}

func NewGetActivityHeatmapUseCase(reader DashboardReader) *GetActivityHeatmapUseCase {
	return &GetActivityHeatmapUseCase{reader: reader}
}

func (uc *GetActivityHeatmapUseCase) Execute(activityTypes []business.ActivityType) map[string]map[string]dashboardDomain.ActivityHeatmapDay {
	heatmap := uc.reader.FindActivityHeatmap(activityTypes...)
	if heatmap == nil {
		return map[string]map[string]dashboardDomain.ActivityHeatmapDay{}
	}

	return heatmap
}

type GetEddingtonNumberUseCase struct {
	reader DashboardReader
}

func NewGetEddingtonNumberUseCase(reader DashboardReader) *GetEddingtonNumberUseCase {
	return &GetEddingtonNumberUseCase{reader: reader}
}

func (uc *GetEddingtonNumberUseCase) Execute(activityTypes []business.ActivityType) business.EddingtonNumber {
	return uc.reader.FindEddingtonNumber(activityTypes...)
}
