package application

import "testing"

type healthReaderStub struct {
	details map[string]any
}

func (stub *healthReaderStub) FindCacheHealthDetails() map[string]any {
	return stub.details
}

func TestGetCacheHealthDetailsUseCase_Execute_ReturnsDetails(t *testing.T) {
	// GIVEN
	reader := &healthReaderStub{
		details: map[string]any{"ok": true},
	}
	useCase := NewGetCacheHealthDetailsUseCase(reader)

	// WHEN
	result := useCase.Execute()

	// THEN
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got %+v", result)
	}
}

func TestGetCacheHealthDetailsUseCase_Execute_ReturnsEmptyMapOnNilReaderResult(t *testing.T) {
	// GIVEN
	reader := &healthReaderStub{details: nil}
	useCase := NewGetCacheHealthDetailsUseCase(reader)

	// WHEN
	result := useCase.Execute()

	// THEN
	if result == nil {
		t.Fatal("expected non-nil map")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %d key(s)", len(result))
	}
}
