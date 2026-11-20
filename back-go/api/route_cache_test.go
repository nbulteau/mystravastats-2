package api

import (
	"testing"
	"time"
)

func TestDeleteGeneratedRouteIfExpired_PreservesEntryRenewedAfterStaleRead(t *testing.T) {
	// GIVEN: a read has observed an expired value, then a concurrent generation
	// replaces it with a fresh value before the stale reader obtains the write lock.
	now := time.Now()
	routeID := "generated-osm-cache-race-test"
	observed := generatedRouteCacheEntry{
		Name:      "expired route",
		Points:    [][]float64{{48.13, -1.63}, {48.14, -1.63}},
		ExpiresAt: now.Add(-time.Minute),
	}
	fresh := generatedRouteCacheEntry{
		Name:      "refreshed route",
		Points:    [][]float64{{48.13, -1.63}, {48.15, -1.63}},
		ExpiresAt: now.Add(time.Hour),
	}

	generatedRouteCache.mu.Lock()
	previous, hadPrevious := generatedRouteCache.items[routeID]
	generatedRouteCache.items[routeID] = fresh
	generatedRouteCache.mu.Unlock()
	t.Cleanup(func() {
		generatedRouteCache.mu.Lock()
		defer generatedRouteCache.mu.Unlock()
		if hadPrevious {
			generatedRouteCache.items[routeID] = previous
		} else {
			delete(generatedRouteCache.items, routeID)
		}
	})

	// WHEN
	deleted := deleteGeneratedRouteIfExpired(routeID, observed, now)

	// THEN
	if deleted {
		t.Fatal("expected stale cleanup not to delete the refreshed route")
	}
	got, found := getGeneratedRouteFromCache(routeID)
	if !found {
		t.Fatal("expected refreshed route to remain in cache")
	}
	if got.Name != fresh.Name || !got.ExpiresAt.Equal(fresh.ExpiresAt) {
		t.Fatalf("expected refreshed entry %q to remain, got %+v", fresh.Name, got)
	}
}
