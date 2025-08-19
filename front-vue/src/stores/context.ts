// noinspection UnnecessaryLocalVariableJS

import { defineStore } from 'pinia'
import { ErrorService } from "@/services/error.service";
import { type Toast } from '@/models/toast.model'
import type { Statistics } from '@/models/statistics.model';
import type { Activity } from '@/models/activity.model';
import { EddingtonNumber } from '@/models/eddington-number.model';
import type { BadgeCheckResult } from '@/models/badge-check-result.model';
import { DashboardData } from '@/models/dashboard-data.model';


export const useContextStore = defineStore('context', {
    state(): {
        athleteDisplayName: string
        currentYear: string
        currentActivityType: string,

        statistics: Statistics[]
        activities: Activity[],
        gpxCoordinates: number[][][],
        distanceByMonths: Map<string, number>[],
        elevationByMonths: Map<string, number>[],
        averageSpeedByMonths: Map<string, number>[],
        distanceByWeeks: Map<string, number>[],
        elevationByWeeks: Map<string, number>[],
        cumulativeDistancePerYear: Map<string, Map<string, number>>,
        cumulativeElevationPerYear: Map<string, Map<string, number>>,
        eddingtonNumber: EddingtonNumber,
        dashboardData: DashboardData,
        generalBadgesCheckResults: BadgeCheckResult[],
        famousClimbBadgesCheckResults: BadgeCheckResult[],


        currentView: 'statistics' | 'activities' | 'activity' | 'map' | 'badges' | 'charts' | 'dashboard'
        toasts: any[]
    } {
        return {
            athleteDisplayName: '',
            currentYear: new Date().getFullYear().toString(),
            currentActivityType: 'Commute_GravelRide_MountainBikeRide_Ride_VirtualRide', // Default activity types (all cycling types)

            statistics: [],
            activities: [],
            gpxCoordinates: [],
            distanceByMonths: [],
            elevationByMonths: [],
            averageSpeedByMonths: [],
            distanceByWeeks: [],
            elevationByWeeks: [],
            eddingtonNumber: new EddingtonNumber(),
            dashboardData: new DashboardData({},  {},  {},  {},  {},  {},  {},  {},  {},  {},  {},  {},  {}, []),
            cumulativeDistancePerYear: new Map<string, Map<string, number>>(),
            cumulativeElevationPerYear: new Map<string, Map<string, number>>(),
            generalBadgesCheckResults: [],
            famousClimbBadgesCheckResults: [],

            currentView: 'statistics',
            toasts: [],
        }
    },
    getters: {
        athleteName: (state) => state.athleteDisplayName,
        hasBadges: (state) => state.generalBadgesCheckResults.length > 0 && state.famousClimbBadgesCheckResults.length > 0
    },
    actions: {
        url(path: string): string {
            const url = `/api/${path}?activityType=${this.currentActivityType}`
            if (this.currentYear === "All years") {
                return url
            }
            return `${url}&year=${this.currentYear}`
        },
        async fetchAthlete() {
            const response = await fetch(`/api/athletes/me`)
            if (!response.ok) {
                await ErrorService.catchError(response)
            }
            const data = await response.json()
            this.athleteDisplayName = `${data["firstname"]} ${data["lastname"]}`
        },
        async fetchStatistics() {
            const statistics = await fetch(this.url("statistics"))
                .then(response => response.json())
            this.statistics = statistics
        },
        async fetchActivities() {
            const activities = await fetch(this.url("activities"))
                .then(response => response.json())
            this.activities = activities
        },
        async fetchGPXCoordinates() {
            const gpxCoordinates = await fetch(this.url("maps/gpx"))
                .then(response => response.json())
            this.gpxCoordinates = gpxCoordinates
        },
        async fetchDistanceByMonths() {
            const distanceByMonths = await fetch(this.url("charts/distance-by-period") + '&period=MONTHS').
                then(response => response.json())
            this.distanceByMonths = distanceByMonths;
        },
        async fetchElevationByMonths() {
            const elevationByMonths = await fetch(this.url("charts/elevation-by-period") + '&period=MONTHS')
                .then(response => response.json())
            this.elevationByMonths = elevationByMonths;
        },
        async fetchAverageSpeedByMonths() {
            const averageSpeedByMonths = await fetch(this.url("charts/average-speed-by-period") + '&period=MONTHS')
                .then(response => response.json())
            this.averageSpeedByMonths = averageSpeedByMonths;
        },
        async fetchDistanceByWeeks() {
            const distanceByWeeks = await fetch(this.url("charts/distance-by-period") + '&period=WEEKS')
                .then(response => response.json())
            this.distanceByWeeks = distanceByWeeks;
        },
        async fetchElevationByWeeks() {
            const elevationByWeeks = await fetch(this.url("charts/elevation-by-period") + '&period=WEEKS')
                .then(response => response.json())
            this.elevationByWeeks = elevationByWeeks;
        },
        async fetchCumulativeDataPerYear() {
            const data = await fetch(this.url("dashboard/cumulative-data-per-year"))
                .then(response => response.json())

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
            const eddingtonNumber = await fetch(this.url("dashboard/eddington-number"))
                .then(response => response.json())
            this.eddingtonNumber = eddingtonNumber;
        },
        async fetchDashboardData() {
            const dashboardData = await fetch(this.url("dashboard"))
                .then(response => response.json())
            this.dashboardData = dashboardData;
        },
        async fetchBadges() {
            const generalBadgesCheckResults = await fetch(this.url("badges"))
                .then(response => response.json())
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
        async updateData() {
            switch (this.currentView) {
                case 'statistics':
                    await this.fetchStatistics()
                    break
                case 'activities':
                    await this.fetchActivities()
                    break
                case 'map':
                    await this.fetchGPXCoordinates()
                    break
                case 'charts':
                    if (this.currentYear != 'All years') {
                        await this.fetchDistanceByMonths()
                        await this.fetchElevationByMonths()
                        await this.fetchAverageSpeedByMonths()
                        await this.fetchDistanceByWeeks()
                        await this.fetchElevationByWeeks()
                    }
                    break
                case 'dashboard':
                    await this.fetchEddingtonNumber()
                    await this.fetchCumulativeDataPerYear()
                    await this.fetchDashboardData()
                    break
                case 'badges':
                    await this.fetchBadges()
                    break
            }
        },
        updateCurrentView(view: 'statistics' | 'activities' | 'activity' | 'map' | 'badges' | 'charts' | 'dashboard') {
            this.currentView = view
            this.updateData().then(
                () => {
                    console.log(`Data updated for view ${view}`)
                }
            );
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

