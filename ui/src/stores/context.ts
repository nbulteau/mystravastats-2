// noinspection UnnecessaryLocalVariableJS

import { defineStore } from 'pinia'
import { ErrorService } from "@/services/error.service";
import { type Toast } from '@/models/toast.model'
import type { Statistics } from '@/models/statistics.model';
import type { Activity } from '@/models/activity.model';
import { EddingtonNumber } from '@/models/eddington-number.model';
import type { BadgeCheckResult } from '@/models/badge-check-result.model';



export const useContextStore = defineStore('context', {
    state(): {
        athleteDisplayName: string
        currentYear: string
        currentActivity: string

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
        generalBadgesCheckResults: BadgeCheckResult[],
        famousClimbBadgesCheckResults: BadgeCheckResult[],


        currentView: 'statistics' | 'activities' | 'activity' | 'map' | 'badges' | 'charts'
        toasts: any[]
    } {
        return {
            athleteDisplayName: '',
            currentYear: new Date().getFullYear().toString(),
            currentActivity: 'Ride',

            statistics: [],
            activities: [],
            gpxCoordinates: [],
            distanceByMonths: [],
            elevationByMonths: [],
            averageSpeedByMonths: [],
            distanceByWeeks: [],
            elevationByWeeks: [],
            eddingtonNumber: new EddingtonNumber(),
            cumulativeDistancePerYear: new Map<string, Map<string, number>>(),
            cumulativeElevationPerYear: new Map<string, Map<string, number>>(),
            generalBadgesCheckResults: [],
            famousClimbBadgesCheckResults: [],

            currentView: 'statistics',
            toasts: [],
        }
    },
    getters: {
        athleteName: (state) => state.athleteDisplayName
    },
    actions: {
        url(path: string): string {
            const url = `http://localhost:8080/api/${path}?activityType=${this.currentActivity}`
            if (this.currentYear === "All years") {
                return url
            }
            return `${url}&year=${this.currentYear}`
        },
        async fetchAthlete() {
            const response = await fetch(`http://localhost:8080/api/athletes/me`)
            if (!response.ok) {
                await ErrorService.catchError(response)
            }
            const data = await response.json()
            this.athleteDisplayName = `${data["firstname"]} ${data["lastname"]}`
        },
        async fetchStatistics() {
            // noinspection UnnecessaryLocalVariableJS
            const statitstics = await fetch(this.url("statistics"))
                .then(response => response.json())
            this.statistics = statitstics
        },
        async fetchActivities() {
            // noinspection UnnecessaryLocalVariableJS
            const activities = await fetch(this.url("activities"))
                .then(response => response.json())
            this.activities = activities
        },
        async fetchGPXCoordinates() {
            // noinspection UnnecessaryLocalVariableJS
            const gpxCoordinates = await fetch(this.url("maps/gpx"))
                .then(response => response.json())
            this.gpxCoordinates = gpxCoordinates
        },
        async fetchDistanceByMonths() {
            // noinspection UnnecessaryLocalVariableJS
            const distanceByMonths = await fetch(this.url("charts/distance-by-period") + '&period=MONTHS').
                then(response => response.json())
            this.distanceByMonths = distanceByMonths;
        },
        async fetchElevationByMonths() {
            // noinspection UnnecessaryLocalVariableJS
            const elevationByMonths = await fetch(this.url("charts/elevation-by-period") + '&period=MONTHS')
                .then(response => response.json())
            this.elevationByMonths = elevationByMonths;
        },
        async fetchAverageSpeedByMonths() {
            // noinspection UnnecessaryLocalVariableJS
            const averageSpeedByMonths = await fetch(this.url("charts/average-speed-by-period") + '&period=MONTHS')
                .then(response => response.json())
            this.averageSpeedByMonths = averageSpeedByMonths;
        },
        async fetchDistanceByWeeks() {
            // noinspection UnnecessaryLocalVariableJS
            const distanceByWeeks = await fetch(this.url("charts/distance-by-period") + '&period=WEEKS')
                .then(response => response.json())
            this.distanceByWeeks = distanceByWeeks;
        },
        async fetchElevationByWeeks() {
            // noinspection UnnecessaryLocalVariableJS
            const elevationByWeeks = await fetch(this.url("charts/elevation-by-period") + '&period=WEEKS')
                .then(response => response.json())
            this.elevationByWeeks = elevationByWeeks;
        },
        async fetchCumulativeDistancePerYear() {
            const data = await fetch(
                `http://localhost:8080/api/charts/cumulative-distance-per-year?activityType=${this.currentActivity}`,
            ).then(response => response.json())

            // Convert the fetched data to a Map<string, Map<string, number>>
            const cumulativeDistancePerYear = new Map<string, Map<string, number>>();

            for (const year in data) {
                if (Object.prototype.hasOwnProperty.call(data, year)) {
                    const daysData = new Map<string, number>();
                    for (const datumKey in data[year]) {
                        if (Object.prototype.hasOwnProperty.call(data[year], datumKey)) {
                            daysData.set(datumKey, data[year][datumKey]);
                        }
                    }
                    cumulativeDistancePerYear.set(year, daysData);
                }
            }
            this.cumulativeDistancePerYear = cumulativeDistancePerYear;
        },
        async fetchCumulativeElevationPerYear() {
            const data = await fetch(
                `http://localhost:8080/api/charts/cumulative-elevation-per-year?activityType=${this.currentActivity}`,
            ).then(response => response.json())

            // Convert the fetched data to a Map<string, Map<string, number>>
            const cumulativeElevationPerYear = new Map<string, Map<string, number>>();

            for (const year in data) {
                if (Object.prototype.hasOwnProperty.call(data, year)) {
                    const daysData = new Map<string, number>();
                    for (const datumKey in data[year]) {
                        if (Object.prototype.hasOwnProperty.call(data[year], datumKey)) {
                            daysData.set(datumKey, data[year][datumKey]);
                        }
                    }
                    cumulativeElevationPerYear.set(year, daysData);
                }
            }
            this.cumulativeElevationPerYear = cumulativeElevationPerYear;
        },
        async fetchEddingtonNumber() {
            // noinspection UnnecessaryLocalVariableJS
            const eddingtonNumber = await fetch(this.url("charts/eddington-number"))
                .then(response => response.json())
            this.eddingtonNumber = eddingtonNumber;
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
            this.currentActivity = activityType
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
                    await this.fetchEddingtonNumber()
                    await this.fetchCumulativeDistancePerYear()
                    await this.fetchCumulativeElevationPerYear()
                    if (this.currentYear != 'All years') {
                        await this.fetchDistanceByMonths()
                        await this.fetchElevationByMonths()
                        await this.fetchAverageSpeedByMonths()
                        await this.fetchDistanceByWeeks()
                        await this.fetchElevationByWeeks()
                    }
                    break
                case 'badges':
                    await this.fetchBadges()
                    break
            }
        },
        updateCurrentView(view: 'statistics' | 'activities' | 'activity' | 'map' | 'badges' | 'charts') {
            this.currentView = view
            this.updateData();
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

