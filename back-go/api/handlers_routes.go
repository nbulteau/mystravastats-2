package api

import (
	"encoding/json"
	"fmt"
	"log"
	"mystravastats/api/dto"
	routesDomain "mystravastats/internal/routes/domain"
	"mystravastats/internal/shared/domain/business"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func getRouteRecommendationsByActivityType(writer http.ResponseWriter, request *http.Request) {
	year, activityTypes, req, err := parseRouteExplorerRequestParams(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	explorer := getContainer().getRouteExplorerUseCase.Execute(year, req, activityTypes)
	explorerDto := dto.ToRouteExplorerResultDto(explorer)
	if err := writeJSON(writer, http.StatusOK, explorerDto); err != nil {
		log.Printf("failed to write routes explorer response: %v", err)
		writeInternalServerError(writer, "Failed to encode routes explorer response")
	}
}

func getRouteRecommendationGPXByActivityType(writer http.ResponseWriter, request *http.Request) {
	year, activityTypes, req, err := parseRouteExplorerRequestParams(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	routeID := strings.TrimSpace(request.URL.Query().Get("routeId"))
	if routeID == "" {
		writeBadRequest(writer, "Invalid request parameters", "routeId is required")
		return
	}

	explorer := getContainer().getRouteExplorerUseCase.Execute(year, req, activityTypes)
	name, points, found := findRouteForGPXExport(explorer, routeID)
	if !found {
		writeNotFound(writer, "Route not found", fmt.Sprintf("No route found for routeId=%s with current filters", routeID))
		return
	}

	gpxPayload, err := buildRouteGPX(name, points)
	if err != nil {
		writeBadRequest(writer, "Invalid route geometry", err.Error())
		return
	}

	fileName := sanitizeRouteFileName(routeID)
	if fileName == "" {
		fileName = "route"
	}

	writer.Header().Set("Content-Type", "application/gpx+xml; charset=utf-8")
	writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.gpx\"", fileName))
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write([]byte(gpxPayload)); err != nil {
		log.Printf("failed to write route gpx response: %v", err)
	}
}

func generateTargetRoutesByActivityType(writer http.ResponseWriter, request *http.Request) {
	year, activityTypes, err := parseRouteGenerationFilters(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	payload, err := parseGenerateTargetRoutesPayload(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request body", err.Error())
		return
	}
	if err := validateGenerateTargetRoutesPayload(payload); err != nil {
		writeBadRequest(writer, "Invalid request body", err.Error())
		return
	}

	routeType := normalizeGenerateRouteType(payload.RouteType)
	targetMode := normalizeGenerateTargetMode(payload.GenerationMode)
	startDirection := normalizeGenerateStartDirection(payload.StartDirection)
	directionStrict := targetMode == "AUTOMATIC" && isUndefinedGenerateStartDirection(payload.StartDirection)
	if targetMode == "CUSTOM" {
		startDirection = ""
		directionStrict = false
	}
	// Backtracking profile is native ULTRA and is no longer user-configurable.
	const strictBacktracking = true
	variantCount := normalizeGenerateVariantCount(payload.VariantCount)
	preferredStart := &routesDomain.Coordinates{
		Lat: payload.StartPoint.Lat,
		Lng: payload.StartPoint.Lng,
	}

	req := routesDomain.RouteExplorerRequest{
		DistanceTargetKm:    &payload.DistanceTarget,
		ElevationTargetM:    payload.ElevationTarget,
		StartPoint:          preferredStart,
		StartDirection:      optionalNonEmptyString(startDirection),
		DirectionStrict:     optionalBool(directionStrict),
		StrictBacktracking:  optionalBool(strictBacktracking),
		BacktrackingProfile: optionalNonEmptyString(nativeBacktrackingProfile),
		TargetMode:          optionalNonEmptyString(targetMode),
		CustomWaypoints:     toRouteCoordinates(payload.CustomWaypoints),
		RouteType:           optionalNonEmptyString(routeType),
		Limit:               variantCount,
	}

	result := getContainer().getRouteExplorerUseCase.Execute(year, req, activityTypes)
	response := buildTargetGeneratedRoutesResponse(
		result,
		payload.DistanceTarget,
		payload.ElevationTarget,
		routeType,
		startDirection,
		directionStrict,
		strictBacktracking,
		targetMode,
		variantCount,
	)
	cacheGeneratedRoutes(response.Routes)

	if err := writeJSON(writer, http.StatusOK, response); err != nil {
		log.Printf("failed to write generated target routes response: %v", err)
		writeInternalServerError(writer, "Failed to encode generated routes response")
	}
}

func generateShapeRoutesByActivityType(writer http.ResponseWriter, request *http.Request) {
	year, activityTypes, err := parseRouteGenerationFilters(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	payload, err := parseGenerateShapeRoutesPayload(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request body", err.Error())
		return
	}
	if err := validateGenerateShapeRoutesPayload(payload); err != nil {
		writeBadRequest(writer, "Invalid request body", err.Error())
		return
	}

	routeType := normalizeGenerateRouteType(payload.RouteType)
	variantCount := normalizeGenerateVariantCount(payload.VariantCount)
	shapeFilter := inferShapeFilter(payload.ShapeInputType, payload.ShapeData)
	var preferredStart *routesDomain.Coordinates
	if payload.StartPoint != nil {
		preferredStart = &routesDomain.Coordinates{
			Lat: payload.StartPoint.Lat,
			Lng: payload.StartPoint.Lng,
		}
	}

	req := routesDomain.RouteExplorerRequest{
		DistanceTargetKm: payload.DistanceTarget,
		ElevationTargetM: payload.ElevationTarget,
		StartPoint:       preferredStart,
		RouteType:        optionalNonEmptyString(routeType),
		Limit:            variantCount,
		Shape:            optionalNonEmptyString(shapeFilter),
		ShapePolyline:    optionalNonEmptyString(strings.TrimSpace(payload.ShapeData)),
		IncludeRemix:     true,
	}
	result := getContainer().getRouteExplorerUseCase.Execute(year, req, activityTypes)
	response := buildShapeGeneratedRoutesResponse(
		result,
		payload.DistanceTarget,
		payload.ElevationTarget,
		routeType,
		variantCount,
	)
	cacheGeneratedRoutes(response.Routes)

	if err := writeJSON(writer, http.StatusOK, response); err != nil {
		log.Printf("failed to write generated shape routes response: %v", err)
		writeInternalServerError(writer, "Failed to encode generated routes response")
	}
}

func getGeneratedRouteGPXByID(writer http.ResponseWriter, request *http.Request) {
	routeID := strings.TrimSpace(mux.Vars(request)["routeId"])
	if routeID == "" {
		writeBadRequest(writer, "Invalid request parameters", "routeId is required")
		return
	}

	entry, found := getGeneratedRouteFromCache(routeID)
	if !found {
		writeNotFound(writer, "Route not found", fmt.Sprintf("No generated route found for routeId=%s", routeID))
		return
	}

	gpxPayload, err := buildRouteGPX(entry.Name, entry.Points)
	if err != nil {
		writeBadRequest(writer, "Invalid route geometry", err.Error())
		return
	}

	fileName := sanitizeRouteFileName(routeID)
	if fileName == "" {
		fileName = "route"
	}

	writer.Header().Set("Content-Type", "application/gpx+xml; charset=utf-8")
	writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.gpx\"", fileName))
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write([]byte(gpxPayload)); err != nil {
		log.Printf("failed to write generated route gpx response: %v", err)
	}
}

// parseGenerateTargetRoutesPayload decodes a generateTargetRoutesPayload from the request body.
func parseGenerateTargetRoutesPayload(request *http.Request) (generateTargetRoutesPayload, error) {
	defer func() { _ = request.Body.Close() }()
	var payload generateTargetRoutesPayload
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		return generateTargetRoutesPayload{}, fmt.Errorf("target payload is invalid")
	}
	return payload, nil
}

// parseGenerateShapeRoutesPayload decodes a generateShapeRoutesPayload from the request body.
func parseGenerateShapeRoutesPayload(request *http.Request) (generateShapeRoutesPayload, error) {
	defer func() { _ = request.Body.Close() }()
	var payload generateShapeRoutesPayload
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		return generateShapeRoutesPayload{}, fmt.Errorf("shape payload is invalid")
	}
	return payload, nil
}

// parseRouteGenerationFilters extracts year and activity types for route generation endpoints.
func parseRouteGenerationFilters(request *http.Request) (*int, []business.ActivityType, error) {
	year, err := getYearParam(request)
	if err != nil {
		return nil, nil, err
	}
	activityTypeRaw := strings.TrimSpace(request.URL.Query().Get("activityType"))
	if activityTypeRaw == "" {
		return year, defaultRouteGenerationActivityTypes(), nil
	}
	activityTypes, err := getActivityTypeParam(request)
	if err != nil {
		return nil, nil, err
	}
	return year, activityTypes, nil
}
