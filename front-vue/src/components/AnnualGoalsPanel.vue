<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import type {
  AnnualGoalMetric,
  AnnualGoalProgress,
  AnnualGoals,
  AnnualGoalStatus,
  AnnualGoalTargets,
} from "@/models/annual-goals.model";
import { emptyAnnualGoalTargets } from "@/models/annual-goals.model";

const props = defineProps<{
  annualGoals: AnnualGoals;
  selectedYear: string;
  activityType: string;
  saving: boolean;
  error: string | null;
}>();

const emit = defineEmits<{
  save: [targets: AnnualGoalTargets];
}>();

const metricOrder: AnnualGoalMetric[] = [
  "DISTANCE_KM",
  "ELEVATION_METERS",
  "MOVING_TIME_SECONDS",
  "ACTIVITIES",
  "ACTIVE_DAYS",
  "EDDINGTON",
];

const metricLabels: Record<AnnualGoalMetric, string> = {
  DISTANCE_KM: "Distance",
  ELEVATION_METERS: "Dénivelé",
  MOVING_TIME_SECONDS: "Temps",
  ACTIVITIES: "Sorties",
  ACTIVE_DAYS: "Jours actifs",
  EDDINGTON: "Eddington",
};

const statusLabels: Record<AnnualGoalStatus, string> = {
  NOT_SET: "À définir",
  AHEAD: "En avance",
  ON_TRACK: "Juste",
  BEHIND: "En retard",
};

const targetInputs = reactive<Record<AnnualGoalMetric, string>>({
  DISTANCE_KM: "",
  ELEVATION_METERS: "",
  MOVING_TIME_SECONDS: "",
  ACTIVITIES: "",
  ACTIVE_DAYS: "",
  EDDINGTON: "",
});

const isYearSelected = computed(() => props.selectedYear !== "All years");

const rows = computed(() => metricOrder.map((metric) => {
  return props.annualGoals.progress.find((item) => item.metric === metric) ?? fallbackProgress(metric);
}));

watch(
  () => props.annualGoals.targets,
  (targets) => {
    targetInputs.DISTANCE_KM = inputValue(targets.distanceKm);
    targetInputs.ELEVATION_METERS = inputValue(targets.elevationMeters);
    targetInputs.MOVING_TIME_SECONDS = inputValue(
      targets.movingTimeSeconds === null ? null : (targets.movingTimeSeconds ?? 0) / 3600,
    );
    targetInputs.ACTIVITIES = inputValue(targets.activities);
    targetInputs.ACTIVE_DAYS = inputValue(targets.activeDays);
    targetInputs.EDDINGTON = inputValue(targets.eddington);
  },
  { immediate: true, deep: true },
);

function fallbackProgress(metric: AnnualGoalMetric): AnnualGoalProgress {
  return {
    metric,
    label: metricLabels[metric],
    unit: metric === "DISTANCE_KM" ? "km" : "",
    current: 0,
    target: 0,
    progressPercent: 0,
    expectedProgressPercent: 0,
    projectedEndOfYear: 0,
    requiredPace: 0,
    requiredPaceUnit: "",
    status: "NOT_SET",
  };
}

function inputValue(value: number | null | undefined): string {
  if (value === null || value === undefined || value <= 0) {
    return "";
  }
  return Number.isInteger(value) ? value.toString() : value.toFixed(1);
}

function parsePositiveNumber(metric: AnnualGoalMetric): number | null {
  const value = Number.parseFloat(targetInputs[metric]);
  if (!Number.isFinite(value) || value <= 0) {
    return null;
  }
  return value;
}

function buildTargets(): AnnualGoalTargets {
  const distanceKm = parsePositiveNumber("DISTANCE_KM");
  const elevationMeters = parsePositiveNumber("ELEVATION_METERS");
  const movingTimeHours = parsePositiveNumber("MOVING_TIME_SECONDS");
  const activities = parsePositiveNumber("ACTIVITIES");
  const activeDays = parsePositiveNumber("ACTIVE_DAYS");
  const eddington = parsePositiveNumber("EDDINGTON");

  return {
    ...emptyAnnualGoalTargets(),
    distanceKm,
    elevationMeters: elevationMeters === null ? null : Math.round(elevationMeters),
    movingTimeSeconds: movingTimeHours === null ? null : Math.round(movingTimeHours * 3600),
    activities: activities === null ? null : Math.round(activities),
    activeDays: activeDays === null ? null : Math.round(activeDays),
    eddington: eddington === null ? null : Math.round(eddington),
  };
}

function saveGoals() {
  emit("save", buildTargets());
}

function inputStep(metric: AnnualGoalMetric): string {
  return metric === "DISTANCE_KM" || metric === "MOVING_TIME_SECONDS" ? "0.1" : "1";
}

function inputUnit(metric: AnnualGoalMetric): string {
  if (metric === "MOVING_TIME_SECONDS") {
    return "h";
  }
  if (metric === "DISTANCE_KM") {
    return "km";
  }
  if (metric === "ELEVATION_METERS") {
    return "m";
  }
  return "";
}

function formatValue(metric: AnnualGoalMetric, value: number): string {
  if (!Number.isFinite(value)) {
    return "-";
  }
  if (metric === "MOVING_TIME_SECONDS") {
    return formatDuration(value);
  }
  if (metric === "DISTANCE_KM") {
    return `${value.toFixed(1)} km`;
  }
  if (metric === "ELEVATION_METERS") {
    return `${Math.round(value).toLocaleString()} m`;
  }
  return Math.round(value).toLocaleString();
}

function formatPace(row: AnnualGoalProgress): string {
  if (row.target <= 0 || row.requiredPace <= 0) {
    return "-";
  }
  if (row.metric === "MOVING_TIME_SECONDS") {
    return `${formatDuration(row.requiredPace)}/j`;
  }
  const value = row.metric === "DISTANCE_KM" ? row.requiredPace.toFixed(1) : Math.ceil(row.requiredPace).toString();
  return `${value} ${inputUnit(row.metric) || row.unit}/j`;
}

function formatProgress(row: AnnualGoalProgress): string {
  if (row.target <= 0) {
    return "-";
  }
  return `${row.progressPercent.toFixed(1)}%`;
}

function progressWidth(row: AnnualGoalProgress): string {
  return `${Math.max(0, Math.min(100, row.progressPercent))}%`;
}

function formatDuration(totalSeconds: number): string {
  const seconds = Math.max(0, totalSeconds);
  const hours = seconds / 3600;
  if (hours >= 1) {
    return `${hours.toFixed(1)} h`;
  }
  return `${Math.round(seconds / 60)} min`;
}
</script>

<template>
  <section class="chart-panel annual-goals">
    <div class="annual-goals__header">
      <div>
        <h3 class="chart-panel__title">
          Objectifs annuels
        </h3>
        <p class="annual-goals__context">
          {{ selectedYear }} · {{ activityType.split("_").join(" + ") }}
        </p>
      </div>
      <button
        type="button"
        class="btn btn-primary btn-sm annual-goals__save"
        :disabled="!isYearSelected || saving"
        @click="saveGoals"
      >
        {{ saving ? "Enregistrement..." : "Enregistrer" }}
      </button>
    </div>

    <div
      v-if="!isYearSelected"
      class="chart-empty annual-goals__empty"
    >
      Choisis une année pour suivre des objectifs annuels.
    </div>

    <div v-else>
      <div
        v-if="error"
        class="annual-goals__error"
      >
        {{ error }}
      </div>

      <div class="annual-goals__table">
        <div class="annual-goals__row annual-goals__row--head">
          <span>Objectif</span>
          <span>Réalisé</span>
          <span>Cible</span>
          <span>Projection</span>
          <span>Rythme</span>
          <span>Statut</span>
        </div>

        <div
          v-for="row in rows"
          :key="row.metric"
          class="annual-goals__row"
        >
          <strong>{{ metricLabels[row.metric] }}</strong>
          <span>{{ formatValue(row.metric, row.current) }}</span>
          <label class="annual-goals__target">
            <input
              v-model="targetInputs[row.metric]"
              class="form-control form-control-sm"
              type="number"
              min="0"
              :step="inputStep(row.metric)"
              :aria-label="`Cible ${metricLabels[row.metric]}`"
            >
            <span>{{ inputUnit(row.metric) }}</span>
          </label>
          <span>{{ formatValue(row.metric, row.projectedEndOfYear) }}</span>
          <span>{{ formatPace(row) }}</span>
          <span
            class="annual-goals__status"
            :class="`annual-goals__status--${row.status.toLowerCase().replace('_', '-')}`"
          >
            {{ statusLabels[row.status] }}
          </span>
          <div class="annual-goals__progress">
            <span :style="{ width: progressWidth(row) }" />
          </div>
          <small class="annual-goals__progress-label">{{ formatProgress(row) }}</small>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.annual-goals {
  padding-bottom: 12px;
}

.annual-goals__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.annual-goals__context {
  margin: 2px 0 0;
  color: var(--ms-text-muted);
  font-size: 0.82rem;
}

.annual-goals__save {
  min-width: 112px;
  white-space: nowrap;
}

.annual-goals__empty {
  margin: 4px 0 0;
}

.annual-goals__error {
  margin-bottom: 10px;
  border: 1px solid #f1b6bf;
  border-radius: 8px;
  padding: 8px 10px;
  color: #8f2438;
  background: #fff0f3;
  font-size: 0.88rem;
}

.annual-goals__table {
  display: grid;
  gap: 6px;
}

.annual-goals__row {
  display: grid;
  grid-template-columns: minmax(96px, 1fr) minmax(96px, 0.9fr) minmax(124px, 0.9fr) minmax(96px, 0.9fr) minmax(88px, 0.8fr) minmax(88px, 0.7fr);
  align-items: center;
  gap: 8px;
  border-top: 1px solid var(--ms-border);
  padding: 7px 0 6px;
  color: var(--ms-text);
  font-size: 0.86rem;
}

.annual-goals__row--head {
  border-top: 0;
  padding-top: 0;
  color: var(--ms-text-muted);
  font-size: 0.74rem;
  font-weight: 800;
  text-transform: uppercase;
}

.annual-goals__target {
  display: grid;
  grid-template-columns: minmax(72px, 1fr) auto;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--ms-text-muted);
}

.annual-goals__target input {
  min-width: 0;
}

.annual-goals__status {
  justify-self: start;
  border-radius: 6px;
  padding: 3px 6px;
  color: var(--ms-text-muted);
  background: #f4f5f8;
  font-weight: 800;
  font-size: 0.76rem;
  white-space: nowrap;
}

.annual-goals__status--ahead {
  color: #176947;
  background: #e5f6ed;
}

.annual-goals__status--on-track {
  color: #7a4b00;
  background: #fff1cc;
}

.annual-goals__status--behind {
  color: #8f2438;
  background: #fff0f3;
}

.annual-goals__progress {
  grid-column: 1 / -2;
  height: 5px;
  border-radius: 999px;
  overflow: hidden;
  background: #edf0f5;
}

.annual-goals__progress span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--ms-primary);
}

.annual-goals__progress-label {
  justify-self: end;
  color: var(--ms-text-muted);
  font-weight: 700;
}

@media (max-width: 980px) {
  .annual-goals__row {
    grid-template-columns: minmax(110px, 1fr) minmax(104px, 1fr) minmax(120px, 1fr);
  }

  .annual-goals__row--head {
    display: none;
  }

  .annual-goals__status {
    justify-self: start;
  }

  .annual-goals__progress {
    grid-column: 1 / -2;
  }
}

@media (max-width: 640px) {
  .annual-goals__header {
    align-items: stretch;
    flex-direction: column;
  }

  .annual-goals__save {
    width: 100%;
  }

  .annual-goals__row {
    grid-template-columns: minmax(0, 1fr);
    gap: 5px;
  }

  .annual-goals__target {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .annual-goals__progress {
    grid-column: 1;
  }

  .annual-goals__progress-label {
    justify-self: start;
  }
}
</style>
