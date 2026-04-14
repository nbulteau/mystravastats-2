import { defineStore } from "pinia";
import { requestJson } from "@/stores/api";
import type { HeartRateZoneSettings } from "@/models/heart-rate-zone.model";

export const useAthleteStore = defineStore("athlete", {
  state: () => ({
    athleteDisplayName: "",
    athleteLoaded: false,
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
      const data = await requestJson<Record<string, string>>("/api/athletes/me");
      this.athleteDisplayName = `${data.firstname ?? ""} ${data.lastname ?? ""}`.trim();
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
  },
});
