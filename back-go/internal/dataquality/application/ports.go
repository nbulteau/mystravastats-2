package application

import "mystravastats/internal/shared/domain/business"

type DataQualityReader interface {
	GetDataQualityReport() business.DataQualityReport
}

type DataQualityWriter interface {
	ExcludeActivityFromStats(activityID int64, reason string) (business.DataQualityReport, error)
	IncludeActivityInStats(activityID int64) (business.DataQualityReport, error)
}

type DataQualityCorrectionWriter interface {
	PreviewCorrection(issueID string) (business.DataQualityCorrectionPreview, error)
	PreviewSafeCorrections() business.DataQualityCorrectionPreview
	ApplyCorrection(issueID string) (business.DataQualityReport, error)
	ApplySafeCorrections() (business.DataQualityReport, error)
	RevertCorrection(correctionID string) (business.DataQualityReport, error)
}
