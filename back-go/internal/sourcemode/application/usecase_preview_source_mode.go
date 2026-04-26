package application

import "mystravastats/internal/shared/domain/business"

type PreviewSourceModeUseCase struct {
	reader SourceModeReader
}

func NewPreviewSourceModeUseCase(reader SourceModeReader) *PreviewSourceModeUseCase {
	return &PreviewSourceModeUseCase{reader: reader}
}

func (uc *PreviewSourceModeUseCase) Execute(request business.SourceModePreviewRequest) business.SourceModePreview {
	return uc.reader.PreviewSourceMode(request)
}
