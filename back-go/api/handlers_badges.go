package api

import (
	"log"
	"mystravastats/api/dto"
	"net/http"
)

// getBadges godoc
// @Summary Get badges
// @Description Returns badges earned or in progress for a given year and activity types
// @Tags badges
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Param badgeSet query string false "Badge set (GENERAL, FAMOUS)"
// @Success 200 {array} dto.BadgeCheckResultDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/badges [get]
func getBadges(writer http.ResponseWriter, request *http.Request) {
	year, activityTypes, err := parseActivityRequestParams(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	badgeSet, err := getBadgeSetParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	badges := getContainer().getBadgesUseCase.Execute(year, badgeSet, activityTypes)

	badgesDto := make([]dto.BadgeCheckResultDto, len(badges))
	for i, badge := range badges {
		badgesDto[i] = dto.ToBadgeCheckResultDto(badge, activityTypes...)
	}

	if err := writeJSON(writer, http.StatusOK, badgesDto); err != nil {
		log.Printf("failed to write badges response: %v", err)
		writeInternalServerError(writer, "Failed to encode badges response")
	}
}
