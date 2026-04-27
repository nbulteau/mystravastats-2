package application

import "mystravastats/internal/shared/domain/business"

type PreviewDataQualityCorrectionUseCase struct {
	writer DataQualityCorrectionWriter
}

func NewPreviewDataQualityCorrectionUseCase(writer DataQualityCorrectionWriter) *PreviewDataQualityCorrectionUseCase {
	return &PreviewDataQualityCorrectionUseCase{writer: writer}
}

func (uc *PreviewDataQualityCorrectionUseCase) Execute(issueID string) (business.DataQualityCorrectionPreview, error) {
	return uc.writer.PreviewCorrection(issueID)
}

type PreviewSafeDataQualityCorrectionsUseCase struct {
	writer DataQualityCorrectionWriter
}

func NewPreviewSafeDataQualityCorrectionsUseCase(writer DataQualityCorrectionWriter) *PreviewSafeDataQualityCorrectionsUseCase {
	return &PreviewSafeDataQualityCorrectionsUseCase{writer: writer}
}

func (uc *PreviewSafeDataQualityCorrectionsUseCase) Execute() business.DataQualityCorrectionPreview {
	return uc.writer.PreviewSafeCorrections()
}

type ApplyDataQualityCorrectionUseCase struct {
	writer DataQualityCorrectionWriter
}

func NewApplyDataQualityCorrectionUseCase(writer DataQualityCorrectionWriter) *ApplyDataQualityCorrectionUseCase {
	return &ApplyDataQualityCorrectionUseCase{writer: writer}
}

func (uc *ApplyDataQualityCorrectionUseCase) Execute(issueID string) (business.DataQualityReport, error) {
	return uc.writer.ApplyCorrection(issueID)
}

type ApplySafeDataQualityCorrectionsUseCase struct {
	writer DataQualityCorrectionWriter
}

func NewApplySafeDataQualityCorrectionsUseCase(writer DataQualityCorrectionWriter) *ApplySafeDataQualityCorrectionsUseCase {
	return &ApplySafeDataQualityCorrectionsUseCase{writer: writer}
}

func (uc *ApplySafeDataQualityCorrectionsUseCase) Execute() (business.DataQualityReport, error) {
	return uc.writer.ApplySafeCorrections()
}

type RevertDataQualityCorrectionUseCase struct {
	writer DataQualityCorrectionWriter
}

func NewRevertDataQualityCorrectionUseCase(writer DataQualityCorrectionWriter) *RevertDataQualityCorrectionUseCase {
	return &RevertDataQualityCorrectionUseCase{writer: writer}
}

func (uc *RevertDataQualityCorrectionUseCase) Execute(correctionID string) (business.DataQualityReport, error) {
	return uc.writer.RevertCorrection(correctionID)
}
