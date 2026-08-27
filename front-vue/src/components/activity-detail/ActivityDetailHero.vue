<script setup lang="ts">
defineProps<{
  activityName: string;
  activityTypeLabel: string;
  activityDateLabel: string;
  commute: boolean;
  activityVersion: "corrected" | "raw";
  activityVersionLabel: string;
  effortCountLabel: string;
  canSelectCorrectedVersion: boolean;
  stravaActivityUrl: string;
}>();

const emit = defineEmits<{
  versionChange: [version: "corrected" | "raw"];
}>();
</script>

<template>
  <section class="detail-hero">
    <div class="detail-hero__content">
      <p class="detail-hero__kicker">{{ activityTypeLabel }}</p>
      <h1 class="detail-hero__title">{{ activityName }}</h1>
      <div class="detail-hero__meta">
        <span class="detail-chip">{{ activityDateLabel }}</span>
        <span v-if="commute" class="detail-chip">Commute</span>
        <span class="detail-chip detail-chip--active">{{ activityVersionLabel }}</span>
        <span class="detail-chip">{{ effortCountLabel }}</span>
      </div>
    </div>
    <div class="detail-hero__actions">
      <div class="detail-version-toggle" aria-label="Activity data version">
        <button
          type="button"
          :class="['btn btn-sm', activityVersion === 'corrected' ? 'btn-primary' : 'btn-outline-secondary']"
          :disabled="!canSelectCorrectedVersion"
          :title="canSelectCorrectedVersion ? 'Show corrected data' : 'No correction applied'"
          @click="emit('versionChange', 'corrected')"
        >
          Corrected
        </button>
        <button
          type="button"
          :class="['btn btn-sm', activityVersion === 'raw' ? 'btn-primary' : 'btn-outline-secondary']"
          @click="emit('versionChange', 'raw')"
        >
          Raw
        </button>
      </div>
      <a
        v-if="stravaActivityUrl"
        :href="stravaActivityUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="btn btn-primary detail-btn-strava"
      >
        Open on Strava
      </a>
    </div>
  </section>
</template>
