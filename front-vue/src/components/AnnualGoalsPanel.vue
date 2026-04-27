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
  "ACTIVITIES",
  "ACTIVE_DAYS",
  "EDDINGTON",
];

const metricLabels: Record<AnnualGoalMetric, string> = {
  DISTANCE_KM: "Distance",
  ELEVATION_METERS: "Dénivelé",
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

const monthLabels = ["Jan", "Fév", "Mar", "Avr", "Mai", "Juin", "Juil", "Aoû", "Sep", "Oct", "Nov", "Déc"];

const targetInputs = reactive<Record<AnnualGoalMetric, string>>({
  DISTANCE_KM: "",
  ELEVATION_METERS: "",
  ACTIVITIES: "",
  ACTIVE_DAYS: "",
  EDDINGTON: "",
});

const isYearSelected = computed(() => props.selectedYear !== "All years");

const rows = computed(() => metricOrder.map((metric) => {
  return normalizeProgress(props.annualGoals.progress.find((item) => item.metric === metric) ?? fallbackProgress(metric));
}));

watch(
  () => props.annualGoals.targets,
  (targets) => {
    targetInputs.DISTANCE_KM = inputValue(targets.distanceKm);
    targetInputs.ELEVATION_METERS = inputValue(targets.elevationMeters);
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
    requiredWeeklyPace: 0,
    last30Days: 0,
    last30DaysWeeklyPace: 0,
    weeklyPaceGap: 0,
    suggestedTarget: null,
    monthly: [],
    status: "NOT_SET",
  };
}

function normalizeProgress(row: AnnualGoalProgress): AnnualGoalProgress {
  return {
    ...row,
    requiredWeeklyPace: row.requiredWeeklyPace ?? 0,
    last30Days: row.last30Days ?? 0,
    last30DaysWeeklyPace: row.last30DaysWeeklyPace ?? 0,
    weeklyPaceGap: row.weeklyPaceGap ?? 0,
    suggestedTarget: row.suggestedTarget ?? null,
    monthly: row.monthly ?? [],
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
  const activities = parsePositiveNumber("ACTIVITIES");
  const activeDays = parsePositiveNumber("ACTIVE_DAYS");
  const eddington = parsePositiveNumber("EDDINGTON");

  return {
    ...emptyAnnualGoalTargets(),
    distanceKm,
    elevationMeters: elevationMeters === null ? null : Math.round(elevationMeters),
    activities: activities === null ? null : Math.round(activities),
    activeDays: activeDays === null ? null : Math.round(activeDays),
    eddington: eddington === null ? null : Math.round(eddington),
  };
}

function saveGoals() {
  emit("save", buildTargets());
}

function inputStep(metric: AnnualGoalMetric): string {
  return metric === "DISTANCE_KM" ? "0.1" : "1";
}

function inputUnit(metric: AnnualGoalMetric): string {
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
  const value = row.metric === "DISTANCE_KM" ? row.requiredPace.toFixed(1) : Math.ceil(row.requiredPace).toString();
  return `${value} ${inputUnit(row.metric) || row.unit}/j`;
}

function formatWeeklyPace(row: AnnualGoalProgress, value: number): string {
  if (row.target <= 0 || value <= 0) {
    return "-";
  }
  const formatted = row.metric === "DISTANCE_KM" ? value.toFixed(1) : Math.ceil(value).toString();
  return `${formatted} ${inputUnit(row.metric) || row.unit}/sem`;
}

function formatCompactValue(metric: AnnualGoalMetric, value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return "-";
  }
  if (metric === "DISTANCE_KM") {
    return value >= 10 ? `${Math.round(value)} km` : `${value.toFixed(1)} km`;
  }
  if (metric === "ELEVATION_METERS") {
    return value >= 1000 ? `${(value / 1000).toFixed(1)}k m` : `${Math.round(value)} m`;
  }
  return Math.round(value).toString();
}

function formatGap(row: AnnualGoalProgress): string {
  if (row.target <= 0) {
    return "-";
  }
  if (row.weeklyPaceGap <= 0) {
    return "OK";
  }
  return formatWeeklyPace(row, row.weeklyPaceGap);
}

function suggestedInputValue(row: AnnualGoalProgress): string {
  if (row.suggestedTarget === null || row.suggestedTarget <= 0) {
    return "";
  }
  return inputValue(row.suggestedTarget);
}

function applySuggestedTarget(row: AnnualGoalProgress) {
  const value = suggestedInputValue(row);
  if (value === "") {
    return;
  }
  targetInputs[row.metric] = value;
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
          <div
            v-if="row.target > 0"
            class="annual-goals__review"
          >
            <span>
              <strong>30 jours</strong>
              {{ formatWeeklyPace(row, row.last30DaysWeeklyPace) }}
            </span>
            <span>
              <strong>À tenir</strong>
              {{ formatWeeklyPace(row, row.requiredWeeklyPace) }}
            </span>
            <span :class="{ 'annual-goals__review-gap--ok': row.weeklyPaceGap <= 0 }">
              <strong>Manque</strong>
              {{ formatGap(row) }}
            </span>
            <button
              v-if="row.suggestedTarget !== null"
              type="button"
              class="btn btn-outline-secondary btn-sm annual-goals__adjust"
              @click="applySuggestedTarget(row)"
            >
              <i class="fa-solid fa-sliders" aria-hidden="true" />
              Ajuster à {{ formatValue(row.metric, row.suggestedTarget) }}
            </button>
          </div>
          <div
            v-if="row.monthly.length > 0"
            class="annual-goals__monthly"
          >
            <span
              v-for="month in row.monthly"
              :key="`${row.metric}-${month.month}`"
              class="annual-goals__month"
              :class="{ 'annual-goals__month--active': month.value > 0 }"
            >
              <strong>{{ monthLabels[month.month - 1] }}</strong>
              <small>{{ formatCompactValue(row.metric, month.value) }}</small>
            </span>
          </div>
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

.annual-goals__review {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  color: var(--ms-text-muted);
  font-size: 0.78rem;
}

.annual-goals__review span {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
}

.annual-goals__review strong {
  color: var(--ms-text);
  font-weight: 800;
}

.annual-goals__review-gap--ok {
  color: #176947;
}

.annual-goals__adjust {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 28px;
  padding: 3px 8px;
  font-size: 0.76rem;
}

.annual-goals__monthly {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: repeat(12, minmax(42px, 1fr));
  gap: 4px;
}

.annual-goals__month {
  display: grid;
  gap: 2px;
  border: 1px solid var(--ms-border);
  border-radius: 6px;
  padding: 4px 3px;
  color: var(--ms-text-muted);
  background: #fafbfe;
  text-align: center;
  min-width: 0;
}

.annual-goals__month strong {
  font-size: 0.66rem;
  line-height: 1;
}

.annual-goals__month small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.66rem;
  line-height: 1.1;
}

.annual-goals__month--active {
  border-color: #b9d8f5;
  color: #1f4f7a;
  background: #eef6ff;
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

  .annual-goals__monthly {
    grid-template-columns: repeat(6, minmax(48px, 1fr));
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

  .annual-goals__review {
    align-items: flex-start;
    flex-direction: column;
  }

  .annual-goals__adjust {
    width: 100%;
    justify-content: center;
  }

  .annual-goals__monthly {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
