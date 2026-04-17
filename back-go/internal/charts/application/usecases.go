package application

import "mystravastats/internal/shared/domain/business"

type GetDistanceByPeriodUseCase struct {
	reader ChartsReader
}

func NewGetDistanceByPeriodUseCase(reader ChartsReader) *GetDistanceByPeriodUseCase {
	return &GetDistanceByPeriodUseCase{reader: reader}
}

func (uc *GetDistanceByPeriodUseCase) Execute(year *int, period business.Period, activityTypes []business.ActivityType) []ChartPeriodPoint {
	result := uc.reader.FindDistanceByPeriod(year, period, activityTypes...)
	if result == nil {
		return []ChartPeriodPoint{}
	}
	return result
}

type GetElevationByPeriodUseCase struct {
	reader ChartsReader
}

func NewGetElevationByPeriodUseCase(reader ChartsReader) *GetElevationByPeriodUseCase {
	return &GetElevationByPeriodUseCase{reader: reader}
}

func (uc *GetElevationByPeriodUseCase) Execute(year *int, period business.Period, activityTypes []business.ActivityType) []ChartPeriodPoint {
	result := uc.reader.FindElevationByPeriod(year, period, activityTypes...)
	if result == nil {
		return []ChartPeriodPoint{}
	}
	return result
}

type GetAverageSpeedByPeriodUseCase struct {
	reader ChartsReader
}

func NewGetAverageSpeedByPeriodUseCase(reader ChartsReader) *GetAverageSpeedByPeriodUseCase {
	return &GetAverageSpeedByPeriodUseCase{reader: reader}
}

func (uc *GetAverageSpeedByPeriodUseCase) Execute(year *int, period business.Period, activityTypes []business.ActivityType) []ChartPeriodPoint {
	result := uc.reader.FindAverageSpeedByPeriod(year, period, activityTypes...)
	if result == nil {
		return []ChartPeriodPoint{}
	}
	return result
}

type GetAverageCadenceByPeriodUseCase struct {
	reader ChartsReader
}

func NewGetAverageCadenceByPeriodUseCase(reader ChartsReader) *GetAverageCadenceByPeriodUseCase {
	return &GetAverageCadenceByPeriodUseCase{reader: reader}
}

func (uc *GetAverageCadenceByPeriodUseCase) Execute(year *int, period business.Period, activityTypes []business.ActivityType) []ChartPeriodPoint {
	result := uc.reader.FindAverageCadenceByPeriod(year, period, activityTypes...)
	if result == nil {
		return []ChartPeriodPoint{}
	}
	return result
}
