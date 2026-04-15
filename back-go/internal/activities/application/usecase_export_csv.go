package application

import "mystravastats/domain/business"

type ExportActivitiesCSVUseCase struct {
	exporter ActivitiesCSVExporter
}

func NewExportActivitiesCSVUseCase(exporter ActivitiesCSVExporter) *ExportActivitiesCSVUseCase {
	return &ExportActivitiesCSVUseCase{
		exporter: exporter,
	}
}

func (uc *ExportActivitiesCSVUseCase) Execute(year *int, activityTypes []business.ActivityType) string {
	return uc.exporter.ExportCSVByYearAndTypes(year, activityTypes...)
}
