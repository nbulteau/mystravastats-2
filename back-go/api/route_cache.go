package api

import (
	"context"
	"mystravastats/api/dto"
	"strings"
	"sync"
	"time"
)

const (
	defaultRoutesVariantCount = 2
	maxGeneratedVariantCount  = 24
	generatedRouteCacheTTL    = 6 * time.Hour
	defaultTargetMode         = "AUTOMATIC"
	nativeBacktrackingProfile = "ULTRA"
)

type generatedRouteCacheEntry struct {
	Name      string
	Points    [][]float64
	ExpiresAt time.Time
}

var generatedRouteCache = struct {
	mu    sync.RWMutex
	items map[string]generatedRouteCacheEntry
}{
	items: map[string]generatedRouteCacheEntry{},
}

// StartCacheEviction starts a background goroutine that evicts expired entries
// from generatedRouteCache every minute. It stops when ctx is cancelled.
func StartCacheEviction(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				generatedRouteCache.mu.Lock()
				for key, entry := range generatedRouteCache.items {
					if now.After(entry.ExpiresAt) {
						delete(generatedRouteCache.items, key)
					}
				}
				generatedRouteCache.mu.Unlock()
			}
		}
	}()
}

func cacheGeneratedRoutes(routes []dto.GeneratedRouteDto) {
	now := time.Now()
	generatedRouteCache.mu.Lock()
	defer generatedRouteCache.mu.Unlock()

	for _, route := range routes {
		if strings.TrimSpace(route.RouteID) == "" || len(route.PreviewLatLng) < 2 {
			continue
		}
		generatedRouteCache.items[route.RouteID] = generatedRouteCacheEntry{
			Name:      route.Title,
			Points:    route.PreviewLatLng,
			ExpiresAt: now.Add(generatedRouteCacheTTL),
		}
	}
}

func getGeneratedRouteFromCache(routeID string) (generatedRouteCacheEntry, bool) {
	now := time.Now()
	generatedRouteCache.mu.RLock()
	entry, found := generatedRouteCache.items[routeID]
	generatedRouteCache.mu.RUnlock()
	if !found {
		return generatedRouteCacheEntry{}, false
	}

	if entry.ExpiresAt.Before(now) {
		generatedRouteCache.mu.Lock()
		delete(generatedRouteCache.items, routeID)
		generatedRouteCache.mu.Unlock()
		return generatedRouteCacheEntry{}, false
	}
	return entry, true
}
