package application

import "mystravastats/internal/shared/domain/business"

type ApplySourceModeUseCase struct {
	reader SourceModeReader
}

func NewApplySourceModeUseCase(reader SourceModeReader) *ApplySourceModeUseCase {
	return &ApplySourceModeUseCase{reader: reader}
}

func (uc *ApplySourceModeUseCase) Execute(request business.SourceModeApplyRequest) (business.SourceModeApplyResult, error) {
	return uc.reader.ApplySourceMode(request)
}
