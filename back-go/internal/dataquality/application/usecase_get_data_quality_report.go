package application

import "mystravastats/internal/shared/domain/business"

type GetDataQualityReportUseCase struct {
	reader DataQualityReader
}

func NewGetDataQualityReportUseCase(reader DataQualityReader) *GetDataQualityReportUseCase {
	return &GetDataQualityReportUseCase{reader: reader}
}

func (uc *GetDataQualityReportUseCase) Execute() business.DataQualityReport {
	return uc.reader.GetDataQualityReport()
}
