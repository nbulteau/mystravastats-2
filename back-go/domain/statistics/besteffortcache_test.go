package statistics

import (
	"path/filepath"
	"testing"

	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

func TestBestEffortCache_SaveLoadRoundTrip(t *testing.T) {
	ClearBestEffortCache()
	stream := testBestEffortStream()
	supplierCalls := 0

	first := getOrComputeBestEffort(
		42,
		"best-time-distance",
		effortDistanceTarget(1000),
		stream,
		func() *business.ActivityEffort {
			supplierCalls++
			return &business.ActivityEffort{
				Distance: 1000,
				Seconds:  180,
				ActivityShort: business.ActivityShort{
					Id:   42,
					Name: "Warmup effort",
					Type: business.Ride,
				},
			}
		},
	)

	if supplierCalls != 1 {
		t.Fatalf("expected supplier to run once before persistence, got %d", supplierCalls)
	}
	if first == nil {
		t.Fatalf("expected first effort to be computed")
	}

	path := filepath.Join(t.TempDir(), "best-effort-cache.json")
	savedEntries, err := SaveBestEffortCacheToDisk(path)
	if err != nil {
		t.Fatalf("save cache failed: %v", err)
	}
	if savedEntries == 0 {
		t.Fatalf("expected at least one persisted cache entry")
	}

	ClearBestEffortCache()
	loadedEntries, err := LoadBestEffortCacheFromDisk(path)
	if err != nil {
		t.Fatalf("load cache failed: %v", err)
	}
	if loadedEntries != savedEntries {
		t.Fatalf("expected loaded entries %d, got %d", savedEntries, loadedEntries)
	}

	supplierCalls = 0
	second := getOrComputeBestEffort(
		42,
		"best-time-distance",
		effortDistanceTarget(1000),
		stream,
		func() *business.ActivityEffort {
			supplierCalls++
			return &business.ActivityEffort{
				Distance: 1000,
				Seconds:  999,
				ActivityShort: business.ActivityShort{
					Id:   42,
					Name: "Should not be used",
					Type: business.Ride,
				},
			}
		},
	)

	if supplierCalls != 0 {
		t.Fatalf("expected supplier not to run after cache reload, got %d calls", supplierCalls)
	}
	if second == nil || second.Seconds != 180 {
		t.Fatalf("expected cached effort to be reused with 180s, got %+v", second)
	}
}

func TestBestEffortCache_InvalidateByActivityIDs(t *testing.T) {
	ClearBestEffortCache()
	stream := testBestEffortStream()

	getOrComputeBestEffort(1, "best-time-distance", effortDistanceTarget(1000), stream, func() *business.ActivityEffort {
		return &business.ActivityEffort{Distance: 1000, Seconds: 200, ActivityShort: business.ActivityShort{Id: 1, Name: "A", Type: business.Ride}}
	})
	getOrComputeBestEffort(2, "best-time-distance", effortDistanceTarget(1000), stream, func() *business.ActivityEffort {
		return &business.ActivityEffort{Distance: 1000, Seconds: 210, ActivityShort: business.ActivityShort{Id: 2, Name: "B", Type: business.Ride}}
	})

	removed := InvalidateBestEffortCacheByActivityIDs(map[int64]struct{}{1: {}})
	if removed == 0 {
		t.Fatalf("expected invalidation to remove at least one entry")
	}
	if BestEffortCacheSize() == 0 {
		t.Fatalf("expected cache to retain entries for untouched activities")
	}
}

func testBestEffortStream() *strava.Stream {
	return &strava.Stream{
		Distance: strava.DistanceStream{Data: []float64{0, 500, 1000}, OriginalSize: 3},
		Time:     strava.TimeStream{Data: []int{0, 90, 180}, OriginalSize: 3},
		Altitude: &strava.AltitudeStream{Data: []float64{10, 20, 30}, OriginalSize: 3},
	}
}
