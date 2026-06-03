package application

import "mystravastats/internal/shared/domain/business"

type SourceModeReader interface {
	PreviewSourceMode(request business.SourceModePreviewRequest) business.SourceModePreview
	ApplySourceMode(request business.SourceModeApplyRequest) (business.SourceModeApplyResult, error)
}
