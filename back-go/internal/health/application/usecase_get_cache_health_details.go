package application

type GetCacheHealthDetailsUseCase struct {
	reader HealthReader
}

func NewGetCacheHealthDetailsUseCase(reader HealthReader) *GetCacheHealthDetailsUseCase {
	return &GetCacheHealthDetailsUseCase{
		reader: reader,
	}
}

func (uc *GetCacheHealthDetailsUseCase) Execute() map[string]any {
	details := uc.reader.FindCacheHealthDetails()
	if details == nil {
		return map[string]any{}
	}
	return details
}
