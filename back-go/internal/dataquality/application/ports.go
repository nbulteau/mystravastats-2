package application

import "mystravastats/internal/shared/domain/business"

type DataQualityReader interface {
	GetDataQualityReport() business.DataQualityReport
}

type DataQualityWriter interface {
	ExcludeActivityFromStats(activityID int64, reason string) (business.DataQualityReport, error)
	IncludeActivityInStats(activityID int64) (business.DataQualityReport, error)
}
