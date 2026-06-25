import { defineStore } from "pinia";
import { requestJson } from "@/stores/api";
import type { HeartRateZoneSettings } from "@/models/heart-rate-zone.model";
import {
  emptyFtpEstimate,
  emptyAthletePerformanceSettings,
  normalizeFtpEstimate,
  normalizeAthletePerformanceSettings,
  type AthletePerformanceSettings,
  type FtpEstimate,
} from "@/models/athlete-performance-settings.model";

const FTP_ESTIMATE_ACTIVITY_TYPE = "Commute_GravelRide_MountainBikeRide_Ride_VirtualRide";
const FTP_ESTIMATE_WINDOW_DAYS = 180;

export const useAthleteStore = defineStore("athlete", {
  state: () => ({
    athleteDisplayName: "",
    athleteWeight: 0,
    athleteFtp: 0,
    athleteLoaded: false,
    performanceSettings: emptyAthletePerformanceSettings() as AthletePerformanceSettings,
    performanceSettingsLoaded: false,
    ftpEstimate: emptyFtpEstimate() as FtpEstimate,
    ftpEstimateLoaded: false,
    heartRateZoneSettings: {
      maxHr: null,
      thresholdHr: null,
      reserveHr: null,
    } as HeartRateZoneSettings,
    heartRateZoneSettingsLoaded: false,
  }),
  getters: {
    athleteName: (state) => state.athleteDisplayName,
  },
  actions: {
    async fetchAthlete(force = false) {
      if (this.athleteLoaded && !force) {
        return;
      }
      const data = await requestJson<Record<string, unknown>>("/api/athletes/me");
      this.athleteDisplayName = `${data.firstname ?? ""} ${data.lastname ?? ""}`.trim();
      this.athleteWeight = parsePositiveNumber(data.weight);
      this.athleteFtp = parsePositiveNumber(data.ftp);
      this.athleteLoaded = true;
    },
    async fetchHeartRateZoneSettings(force = false) {
      if (this.heartRateZoneSettingsLoaded && !force) {
        return;
      }
      const settings = await requestJson<HeartRateZoneSettings>("/api/athletes/me/heart-rate-zones");
      this.heartRateZoneSettings = settings;
      this.heartRateZoneSettingsLoaded = true;
    },
    async saveHeartRateZoneSettings(settings: HeartRateZoneSettings) {
      const updatedSettings = await requestJson<HeartRateZoneSettings>("/api/athletes/me/heart-rate-zones", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(settings),
      });
      this.heartRateZoneSettings = updatedSettings;
      this.heartRateZoneSettingsLoaded = true;
      return updatedSettings;
    },
    async fetchPerformanceSettings(force = false) {
      if (this.performanceSettingsLoaded && !force) {
        return;
      }
      const settings = await requestJson<AthletePerformanceSettings>("/api/athletes/me/performance-settings");
      this.performanceSettings = normalizeAthletePerformanceSettings(settings);
      this.performanceSettingsLoaded = true;
    },
    async fetchFtpEstimate(force = false) {
      if (this.ftpEstimateLoaded && !force) {
        return;
      }
      const params = new URLSearchParams({
        activityType: FTP_ESTIMATE_ACTIVITY_TYPE,
        days: String(FTP_ESTIMATE_WINDOW_DAYS),
      });
      const estimate = await requestJson<FtpEstimate>(`/api/athletes/me/ftp-estimate?${params.toString()}`);
      this.ftpEstimate = normalizeFtpEstimate(estimate);
      this.ftpEstimateLoaded = true;
    },
    async savePerformanceSettings(settings: AthletePerformanceSettings) {
      const updatedSettings = await requestJson<AthletePerformanceSettings>("/api/athletes/me/performance-settings", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(normalizeAthletePerformanceSettings(settings)),
      });
      this.performanceSettings = normalizeAthletePerformanceSettings(updatedSettings);
      this.performanceSettingsLoaded = true;
      return this.performanceSettings;
    },
  },
});

function parsePositiveNumber(value: unknown): number {
  const parsed = typeof value === "number"
    ? value
    : typeof value === "string"
      ? Number.parseFloat(value)
      : 0;
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}
