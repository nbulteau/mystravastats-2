package application

import "mystravastats/domain/business"

type ListPersonalRecordsTimelineUseCase struct {
	reader PersonalRecordsTimelineReader
}

func NewListPersonalRecordsTimelineUseCase(reader PersonalRecordsTimelineReader) *ListPersonalRecordsTimelineUseCase {
	return &ListPersonalRecordsTimelineUseCase{
		reader: reader,
	}
}

func (uc *ListPersonalRecordsTimelineUseCase) Execute(year *int, metric *string, activityTypes []business.ActivityType) []business.PersonalRecordTimelineEntry {
	timeline := uc.reader.FindPersonalRecordsTimelineByYearMetricAndTypes(year, metric, activityTypes...)
	if timeline == nil {
		return []business.PersonalRecordTimelineEntry{}
	}

	return timeline
}
