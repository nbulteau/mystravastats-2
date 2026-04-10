// noinspection UnnecessaryLocalVariableJS

import { defineStore } from 'pinia'
import { ErrorService } from "@/services/error.service";
import { type Toast } from '@/models/toast.model'
import type { Statistics } from '@/models/statistics.model';
import type { PersonalRecordTimeline } from '@/models/personal-record-timeline.model';
import type { Activity } from '@/models/activity.model';
import { EddingtonNumber } from '@/models/eddington-number.model';
import type { BadgeCheckResult } from '@/models/badge-check-result.model';
import { DashboardData } from '@/models/dashboard-data.model';
import type { SegmentClimbProgression } from '@/models/segment-climb-progression.model';


export const useContextStore = defineStore('context', {
    state(): {
        athleteDisplayName: string
        currentYear: string
        currentActivityType: string,

        statistics: Statistics[]
        personalRecordsTimeline: PersonalRecordTimeline[]
        activities: Activity[],
        gpxCoordinates: number[][][],
        distanceByMonths: Map<string, number>[],
        elevationByMonths: Map<string, number>[],
        averageSpeedByMonths: Map<string, number>[],
        distanceByWeeks: Map<string, number>[],
        elevationByWeeks: Map<string, number>[],
        cadenceByWeeks: Map<string, number>[],
        cumulativeDistancePerYear: Map<string, Map<string, number>>,
        cumulativeElevationPerYear: Map<string, Map<string, number>>,
        eddingtonNumber: EddingtonNumber,
        dashboardData: DashboardData,
        generalBadgesCheckResults: BadgeCheckResult[],
        famousClimbBadgesCheckResults: BadgeCheckResult[],
        segmentClimbProgression: SegmentClimbProgression,
        segmentProgressionMetric: 'TIME' | 'SPEED',
        segmentProgressionTargetType: 'ALL' | 'SEGMENT' | 'CLIMB',
        segmentProgressionSelectedTargetId: number | null,

        currentView: 'statistics' | 'activities' | 'activity' | 'map' | 'badges' | 'charts' | 'dashboard' | 'segments'
        toasts: any[]
    } {
        return {
            athleteDisplayName: '',
            currentYear: new Date().getFullYear().toString(),
            currentActivityType: 'Commute_GravelRide_MountainBikeRide_Ride_VirtualRide', // Default activity types (all cycling types)

            statistics: [],
            personalRecordsTimeline: [],
            activities: [],
            gpxCoordinates: [],
            distanceByMonths: [],
            elevationByMonths: [],
            averageSpeedByMonths: [],
            distanceByWeeks: [],
            elevationByWeeks: [],
            cadenceByWeeks: [],
            eddingtonNumber: new EddingtonNumber(),
            dashboardData: new DashboardData({},  {},  {},  {},  {},  {},  {},  {},  {},  {},  {},  {},  {}, []),
            cumulativeDistancePerYear: new Map<string, Map<string, number>>(),
            cumulativeElevationPerYear: new Map<string, Map<string, number>>(),
            generalBadgesCheckResults: [],
            famousClimbBadgesCheckResults: [],
            segmentClimbProgression: {
                metric: 'TIME',
                targetTypeFilter: 'ALL',
                weatherContextAvailable: false,
                targets: [],
                attempts: [],
            },
            segmentProgressionMetric: 'TIME',
            segmentProgressionTargetType: 'ALL',
            segmentProgressionSelectedTargetId: null,

            currentView: 'statistics',
            toasts: [],
        }
    },
    getters: {
        athleteName: (state) => state.athleteDisplayName,
        hasBadges: (state) => state.generalBadgesCheckResults.length > 0 && state.famousClimbBadgesCheckResults.length > 0
    },
    actions: {
        async fetchJson<T>(url: string): Promise<T> {
            const response = await fetch(url)
            if (!response.ok) {
                await ErrorService.catchError(response)
            }
            return response.json() as Promise<T>
        },
        url(path: string): string {
            const url = `/api/${path}?activityType=${this.currentActivityType}`
            if (this.currentYear === "All years") {
                return url
            }
            return `${url}&year=${this.currentYear}`
        },
        async fetchAthlete() {
            const data = await this.fetchJson<Record<string, string>>(`/api/athletes/me`)
            this.athleteDisplayName = `${data["firstname"]} ${data["lastname"]}`
        },
        async fetchStatistics() {
            const statistics = await this.fetchJson<Statistics[]>(this.url("statistics"))
            this.statistics = statistics
        },
        async fetchPersonalRecordsTimeline() {
            const personalRecordsTimeline = await this.fetchJson<PersonalRecordTimeline[]>(this.url("statistics/personal-records-timeline"))
            this.personalRecordsTimeline = personalRecordsTimeline
        },
        async fetchSegmentClimbProgression() {
            let progressionUrl = `${this.url("statistics/segment-climb-progression")}&metric=${this.segmentProgressionMetric}&targetType=${this.segmentProgressionTargetType}`
            if (this.segmentProgressionSelectedTargetId != null) {
                progressionUrl += `&targetId=${this.segmentProgressionSelectedTargetId}`
            }

            const progression = await this.fetchJson<SegmentClimbProgression>(progressionUrl)
            this.segmentClimbProgression = progression
        },
        async fetchActivities() {
            const activities = await this.fetchJson<Activity[]>(this.url("activities"))
            this.activities = activities
        },
        async fetchGPXCoordinates() {
            const gpxCoordinates = await this.fetchJson<number[][][]>(this.url("maps/gpx"))
            this.gpxCoordinates = gpxCoordinates
        },
        async fetchDistanceByMonths() {
            const distanceByMonths = await this.fetchJson<Map<string, number>[]>(this.url("charts/distance-by-period") + '&period=MONTHS')
            this.distanceByMonths = distanceByMonths;
        },
        async fetchElevationByMonths() {
            const elevationByMonths = await this.fetchJson<Map<string, number>[]>(this.url("charts/elevation-by-period") + '&period=MONTHS')
            this.elevationByMonths = elevationByMonths;
        },
        async fetchAverageSpeedByMonths() {
            const averageSpeedByMonths = await this.fetchJson<Map<string, number>[]>(this.url("charts/average-speed-by-period") + '&period=MONTHS')
            this.averageSpeedByMonths = averageSpeedByMonths;
        },
        async fetchDistanceByWeeks() {
            const distanceByWeeks = await this.fetchJson<Map<string, number>[]>(this.url("charts/distance-by-period") + '&period=WEEKS')
            this.distanceByWeeks = distanceByWeeks;
        },
        async fetchElevationByWeeks() {
            const elevationByWeeks = await this.fetchJson<Map<string, number>[]>(this.url("charts/elevation-by-period") + '&period=WEEKS')
            this.elevationByWeeks = elevationByWeeks;
        },
        async fetchCadenceByWeeks() {
            const cadenceByWeeks = await this.fetchJson<Map<string, number>[]>(this.url("charts/average-cadence-by-period") + '&period=WEEKS')
            this.cadenceByWeeks = cadenceByWeeks;
        },
        async fetchCumulativeDataPerYear() {
            const data = await this.fetchJson<any>(this.url("dashboard/cumulative-data-per-year"))

            // Convert the fetched data to a Map<string, Map<string, number>>
            const cumulativeDistancePerYear = new Map<string, Map<string, number>>();
            for (const year in data.distance) {
                convertToMap(year, data.distance, cumulativeDistancePerYear);
            }
            this.cumulativeDistancePerYear = cumulativeDistancePerYear;

            const cumulativeElevationPerYear = new Map<string, Map<string, number>>();
            for (const year in data.elevation) {
                convertToMap(year, data.elevation, cumulativeElevationPerYear);
            }
            this.cumulativeElevationPerYear = cumulativeElevationPerYear;

            function convertToMap(year: string, data: any, cumulativeDataPerYear: Map<string, Map<string, number>>) {
                const daysData = new Map<string, number>();
                for (const datumKey in data[year]) {
                    if (Object.prototype.hasOwnProperty.call(data[year], datumKey)) {
                        daysData.set(datumKey, data[year][datumKey]);
                    }
                }
                cumulativeDataPerYear.set(year, daysData);
            }
        },
        async fetchEddingtonNumber() {
            const eddingtonNumber = await this.fetchJson<EddingtonNumber>(this.url("dashboard/eddington-number"))
            this.eddingtonNumber = eddingtonNumber;
        },
        async fetchDashboardData() {
            const dashboardData = await this.fetchJson<DashboardData>(this.url("dashboard"))
            this.dashboardData = dashboardData;
        },
        async fetchBadges() {
            const generalBadgesCheckResults = await this.fetchJson<BadgeCheckResult[]>(this.url("badges"))
            this.generalBadgesCheckResults = generalBadgesCheckResults.filter((badgeCheckResult: BadgeCheckResult) => { return !badgeCheckResult.badge.type.endsWith('FamousClimbBadge') });
            this.famousClimbBadgesCheckResults = generalBadgesCheckResults.filter((badgeCheckResult: BadgeCheckResult) => { return badgeCheckResult.badge.type.endsWith('FamousClimbBadge') });
        },
        async updateCurrentYear(currentYear: string) {
            this.currentYear = currentYear
            await this.updateData();
        },
        async updateCurrentActivityType(activityType: string) {
            this.currentActivityType = activityType
            await this.updateData();
        },
        async updateSegmentProgressionMetric(metric: 'TIME' | 'SPEED') {
            this.segmentProgressionMetric = metric
            this.segmentProgressionSelectedTargetId = null
            await this.fetchSegmentClimbProgression()
        },
        async updateSegmentProgressionTargetType(targetType: 'ALL' | 'SEGMENT' | 'CLIMB') {
            this.segmentProgressionTargetType = targetType
            this.segmentProgressionSelectedTargetId = null
            await this.fetchSegmentClimbProgression()
        },
        async updateSegmentProgressionTarget(targetId: number | null) {
            this.segmentProgressionSelectedTargetId = targetId
            await this.fetchSegmentClimbProgression()
        },
        async updateData() {
            switch (this.currentView) {
                case 'statistics':
                    await Promise.all([
                        this.fetchStatistics(),
                        this.fetchPersonalRecordsTimeline(),
                    ])
                    break
                case 'activities':
                    await this.fetchActivities()
                    break
                case 'map':
                    await this.fetchGPXCoordinates()
                    break
                case 'charts':
                    if (this.currentYear != 'All years') {
                        await Promise.all([
                            this.fetchDistanceByMonths(),
                            this.fetchElevationByMonths(),
                            this.fetchAverageSpeedByMonths(),
                            this.fetchDistanceByWeeks(),
                            this.fetchElevationByWeeks(),
                            this.fetchCadenceByWeeks(),
                        ])
                    }
                    break
                case 'dashboard':
                    await Promise.all([
                        this.fetchEddingtonNumber(),
                        this.fetchCumulativeDataPerYear(),
                        this.fetchDashboardData(),
                    ])
                    break
                case 'badges':
                    await this.fetchBadges()
                    break
                case 'segments':
                    await this.fetchSegmentClimbProgression()
                    break
            }
        },
        updateCurrentView(view: 'statistics' | 'activities' | 'activity' | 'map' | 'badges' | 'charts' | 'dashboard' | 'segments') {
            this.currentView = view
            void this.updateData()
        },

        showToast(toast: Toast) {
            this.toasts.push(toast)
            if (toast.timeout) {
                setTimeout(() => {
                    this.removeToast(toast)
                }, toast.timeout)
            }
        },
        removeToast(toast: Toast) {
            this.toasts = this.toasts.filter((t) => t.id !== toast.id)
        },
    }
})
