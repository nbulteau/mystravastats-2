package api

import (
	"sync"

	activitiesApp "mystravastats/internal/activities/application"
	activitiesInfra "mystravastats/internal/activities/infrastructure"
	athleteApp "mystravastats/internal/athlete/application"
	athleteInfra "mystravastats/internal/athlete/infrastructure"
	badgesApp "mystravastats/internal/badges/application"
	badgesInfra "mystravastats/internal/badges/infrastructure"
	chartsApp "mystravastats/internal/charts/application"
	chartsInfra "mystravastats/internal/charts/infrastructure"
	dashboardApp "mystravastats/internal/dashboard/application"
	dashboardInfra "mystravastats/internal/dashboard/infrastructure"
	gearAnalysisApp "mystravastats/internal/gearanalysis/application"
	gearAnalysisInfra "mystravastats/internal/gearanalysis/infrastructure"
	healthApp "mystravastats/internal/health/application"
	healthInfra "mystravastats/internal/health/infrastructure"
	heartrateApp "mystravastats/internal/heartrate/application"
	heartrateInfra "mystravastats/internal/heartrate/infrastructure"
	routesApp "mystravastats/internal/routes/application"
	routesInfra "mystravastats/internal/routes/infrastructure"
	segmentsApp "mystravastats/internal/segments/application"
	segmentsInfra "mystravastats/internal/segments/infrastructure"
	sourceModeApp "mystravastats/internal/sourcemode/application"
	sourceModeInfra "mystravastats/internal/sourcemode/infrastructure"
	statisticsApp "mystravastats/internal/statistics/application"
	statisticsInfra "mystravastats/internal/statistics/infrastructure"
)

type container struct {
	getDetailedActivityUseCase         *activitiesApp.GetDetailedActivityUseCase
	listActivitiesUseCase              *activitiesApp.ListActivitiesUseCase
	exportActivitiesCSVUseCase         *activitiesApp.ExportActivitiesCSVUseCase
	getMapsGPXUseCase                  *activitiesApp.GetMapsGPXUseCase
	getAthleteUseCase                  *athleteApp.GetAthleteUseCase
	listStatisticsUseCase              *statisticsApp.ListStatisticsUseCase
	listPersonalRecordsTimelineUseCase *statisticsApp.ListPersonalRecordsTimelineUseCase
	getSegmentClimbProgressionUseCase  *segmentsApp.GetSegmentClimbProgressionUseCase
	listSegmentsUseCase                *segmentsApp.ListSegmentsUseCase
	listSegmentEffortsUseCase          *segmentsApp.ListSegmentEffortsUseCase
	getSegmentSummaryUseCase           *segmentsApp.GetSegmentSummaryUseCase
	getRouteExplorerUseCase            *routesApp.GetRouteExplorerUseCase
	getHeartRateZoneSettingsUseCase    *heartrateApp.GetHeartRateZoneSettingsUseCase
	updateHeartRateZoneSettingsUseCase *heartrateApp.UpdateHeartRateZoneSettingsUseCase
	getHeartRateZoneAnalysisUseCase    *heartrateApp.GetHeartRateZoneAnalysisUseCase
	getDistanceByPeriodUseCase         *chartsApp.GetDistanceByPeriodUseCase
	getElevationByPeriodUseCase        *chartsApp.GetElevationByPeriodUseCase
	getAverageSpeedByPeriodUseCase     *chartsApp.GetAverageSpeedByPeriodUseCase
	getAverageCadenceByPeriodUseCase   *chartsApp.GetAverageCadenceByPeriodUseCase
	getDashboardDataUseCase            *dashboardApp.GetDashboardDataUseCase
	getCumulativeDataPerYearUseCase    *dashboardApp.GetCumulativeDataPerYearUseCase
	getActivityHeatmapUseCase          *dashboardApp.GetActivityHeatmapUseCase
	getEddingtonNumberUseCase          *dashboardApp.GetEddingtonNumberUseCase
	getAnnualGoalsUseCase              *dashboardApp.GetAnnualGoalsUseCase
	updateAnnualGoalsUseCase           *dashboardApp.UpdateAnnualGoalsUseCase
	getGearAnalysisUseCase             *gearAnalysisApp.GetGearAnalysisUseCase
	getBadgesUseCase                   *badgesApp.GetBadgesUseCase
	getCacheHealthDetailsUseCase       *healthApp.GetCacheHealthDetailsUseCase
	previewSourceModeUseCase           *sourceModeApp.PreviewSourceModeUseCase
}

var (
	containerOnce   sync.Once
	sharedContainer *container
)

func getContainer() *container {
	containerOnce.Do(func() {
		detailedActivityReader := activitiesInfra.NewDetailedActivityServiceAdapter()
		athleteReader := athleteInfra.NewAthleteServiceAdapter()
		badgesReader := badgesInfra.NewBadgesServiceAdapter()
		statisticsReader := statisticsInfra.NewStatisticsServiceAdapter()
		segmentsReader := segmentsInfra.NewSegmentServiceAdapter()
		routingEngine := routesInfra.NewOSMRoutingAdapter()
		routesReader := routesInfra.NewRouteServiceAdapter(routingEngine)
		heartRateReader := heartrateInfra.NewHeartRateServiceAdapter()
		gearAnalysisReader := gearAnalysisInfra.NewGearAnalysisServiceAdapter()
		healthReader := healthInfra.NewHealthServiceAdapter(routingEngine)
		chartsReader := chartsInfra.NewChartsServiceAdapter()
		dashboardReader := dashboardInfra.NewDashboardServiceAdapter()
		sourceModeReader := sourceModeInfra.NewSourceModeServiceAdapter()
		sharedContainer = &container{
			getDetailedActivityUseCase:         activitiesApp.NewGetDetailedActivityUseCase(detailedActivityReader),
			listActivitiesUseCase:              activitiesApp.NewListActivitiesUseCase(detailedActivityReader),
			exportActivitiesCSVUseCase:         activitiesApp.NewExportActivitiesCSVUseCase(detailedActivityReader),
			getMapsGPXUseCase:                  activitiesApp.NewGetMapsGPXUseCase(detailedActivityReader),
			getAthleteUseCase:                  athleteApp.NewGetAthleteUseCase(athleteReader),
			listStatisticsUseCase:              statisticsApp.NewListStatisticsUseCase(statisticsReader),
			listPersonalRecordsTimelineUseCase: statisticsApp.NewListPersonalRecordsTimelineUseCase(statisticsReader),
			getSegmentClimbProgressionUseCase:  segmentsApp.NewGetSegmentClimbProgressionUseCase(segmentsReader),
			listSegmentsUseCase:                segmentsApp.NewListSegmentsUseCase(segmentsReader),
			listSegmentEffortsUseCase:          segmentsApp.NewListSegmentEffortsUseCase(segmentsReader),
			getSegmentSummaryUseCase:           segmentsApp.NewGetSegmentSummaryUseCase(segmentsReader),
			getRouteExplorerUseCase:            routesApp.NewGetRouteExplorerUseCase(routesReader),
			getHeartRateZoneSettingsUseCase:    heartrateApp.NewGetHeartRateZoneSettingsUseCase(heartRateReader),
			updateHeartRateZoneSettingsUseCase: heartrateApp.NewUpdateHeartRateZoneSettingsUseCase(heartRateReader),
			getHeartRateZoneAnalysisUseCase:    heartrateApp.NewGetHeartRateZoneAnalysisUseCase(heartRateReader),
			getDistanceByPeriodUseCase:         chartsApp.NewGetDistanceByPeriodUseCase(chartsReader),
			getElevationByPeriodUseCase:        chartsApp.NewGetElevationByPeriodUseCase(chartsReader),
			getAverageSpeedByPeriodUseCase:     chartsApp.NewGetAverageSpeedByPeriodUseCase(chartsReader),
			getAverageCadenceByPeriodUseCase:   chartsApp.NewGetAverageCadenceByPeriodUseCase(chartsReader),
			getDashboardDataUseCase:            dashboardApp.NewGetDashboardDataUseCase(dashboardReader),
			getCumulativeDataPerYearUseCase:    dashboardApp.NewGetCumulativeDataPerYearUseCase(dashboardReader),
			getActivityHeatmapUseCase:          dashboardApp.NewGetActivityHeatmapUseCase(dashboardReader),
			getEddingtonNumberUseCase:          dashboardApp.NewGetEddingtonNumberUseCase(dashboardReader),
			getAnnualGoalsUseCase:              dashboardApp.NewGetAnnualGoalsUseCase(dashboardReader),
			updateAnnualGoalsUseCase:           dashboardApp.NewUpdateAnnualGoalsUseCase(dashboardReader),
			getGearAnalysisUseCase:             gearAnalysisApp.NewGetGearAnalysisUseCase(gearAnalysisReader),
			getBadgesUseCase:                   badgesApp.NewGetBadgesUseCase(badgesReader),
			getCacheHealthDetailsUseCase:       healthApp.NewGetCacheHealthDetailsUseCase(healthReader),
			previewSourceModeUseCase:           sourceModeApp.NewPreviewSourceModeUseCase(sourceModeReader),
		}
	})

	return sharedContainer
}
