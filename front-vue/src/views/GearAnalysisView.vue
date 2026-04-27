<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { RouterLink } from "vue-router";
import type {
  GearAnalysisItem,
  GearAnalysisPeriodPoint,
  GearKind,
  GearMaintenanceRecord,
  GearMaintenanceRecordRequest,
  GearMaintenanceTask,
} from "@/models/gear-analysis.model";
import { useContextStore } from "@/stores/context";
import { useGearAnalysisStore } from "@/stores/gear-analysis";
import { useUiStore } from "@/stores/ui";
import { ToastTypeEnum } from "@/models/toast.model";

type KindFilter = "ALL" | GearKind | "RETIRED";
type SortMode = "distance" | "lastUsed" | "elevationGain" | "activities";

const contextStore = useContextStore();
const gearAnalysisStore = useGearAnalysisStore();
const uiStore = useUiStore();
const kindFilter = ref<KindFilter>("ALL");
const sortMode = ref<SortMode>("distance");
const maintenanceFormGearId = ref<string | null>(null);
const savingMaintenanceGearId = ref<string | null>(null);
const deletingMaintenanceRecordId = ref<string | null>(null);
const expandedMaintenanceByGearId = reactive<Record<string, boolean>>({});
const maintenanceForm = reactive({
  component: "Chain",
  operation: "",
  date: todayInputValue(),
  distanceKm: "",
  note: "",
});
const maintenanceComponents = [
  { value: "CHAIN", label: "Chain" },
  { value: "CASSETTE", label: "Cassette" },
  { value: "BRAKE_PADS_FRONT", label: "Front brake pads" },
  { value: "BRAKE_PADS_REAR", label: "Rear brake pads" },
  { value: "BRAKE_BLEED", label: "Brake bleed" },
  { value: "TIRES", label: "Tires" },
  { value: "TUBELESS_FRONT", label: "Front tubeless sealant" },
  { value: "TUBELESS_REAR", label: "Rear tubeless sealant" },
  { value: "BOTTOM_BRACKET", label: "Bottom bracket" },
  { value: "BEARINGS", label: "Bearings" },
  { value: "DRIVETRAIN", label: "Drivetrain" },
  { value: "VALVE_CORE_FRONT", label: "Front valve core" },
  { value: "VALVE_CORE_REAR", label: "Rear valve core" },
  { value: "DERAILLEUR_HANGER", label: "Derailleur hanger" },
  { value: "CHAINRING", label: "Chainring" },
  { value: "WHEEL_TRUING", label: "Wheel truing" },
  { value: "FORK_SERVICE", label: "Fork service" },
  { value: "SHOCK_SERVICE", label: "Shock service" },
];

onMounted(() => contextStore.updateCurrentView("gear"));

const analysis = computed(() => gearAnalysisStore.analysis);
const currentYear = computed(() => contextStore.currentYear);
const isLoading = computed(() => gearAnalysisStore.isLoading);
const error = computed(() => gearAnalysisStore.error);
const maintenanceDueCount = computed(() =>
  analysis.value.items.reduce((sum, item) => sum + item.maintenanceTasks.filter((task) => task.status === "DUE" || task.status === "OVERDUE").length, 0),
);
const coveragePercent = computed(() => {
  const total = analysis.value.coverage.totalActivities;
  if (total <= 0) return 0;
  return Math.round((analysis.value.coverage.assignedActivities / total) * 100);
});

const filteredItems = computed(() => {
  const selectedKind = kindFilter.value;
  const items = analysis.value.items.filter((item) => {
    if (selectedKind === "ALL") return true;
    if (selectedKind === "RETIRED") return item.retired;
    return item.kind === selectedKind;
  });

  return [...items].sort((left, right) => {
    if (sortMode.value === "lastUsed") return right.lastUsed.localeCompare(left.lastUsed);
    if (sortMode.value === "elevationGain") return right.elevationGain - left.elevationGain;
    if (sortMode.value === "activities") return right.activities - left.activities;
    return right.distance - left.distance;
  });
});

function formatDistance(value: number): string {
  return `${(value / 1000).toFixed(1)} km`;
}

function formatElevation(value: number): string {
  return `${Math.round(value)} m`;
}

function formatMovingTime(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (hours <= 0) return `${minutes} min`;
  return `${hours} h ${minutes.toString().padStart(2, "0")}`;
}

function formatSpeed(item: GearAnalysisItem): string {
  if (item.averageSpeed <= 0) return "-";
  if (item.kind === "SHOE") {
    const secondsPerKm = 1000 / item.averageSpeed;
    const minutes = Math.floor(secondsPerKm / 60);
    const seconds = Math.round(secondsPerKm % 60);
    return `${minutes}'${seconds.toString().padStart(2, "0")}/km`;
  }
  return `${(item.averageSpeed * 3.6).toFixed(1)} km/h`;
}

function formatDate(value: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value.slice(0, 10);
  return new Intl.DateTimeFormat(navigator.language, {
    day: "2-digit",
    month: "short",
    year: "numeric",
  }).format(date);
}

function gearKindLabel(kind: GearKind): string {
  if (kind === "BIKE") return "Bike";
  if (kind === "SHOE") return "Shoes";
  return "Gear";
}

function maintenanceClass(item: GearAnalysisItem): string {
  return `gear-pill gear-pill--maintenance gear-pill--maintenance-${item.maintenanceStatus.toLowerCase()}`;
}

function maintenanceTaskClass(task: GearMaintenanceTask): string {
  return `maintenance-status maintenance-status--${task.status.toLowerCase()}`;
}

function maintenanceComponentLabel(component: string): string {
  return maintenanceComponents.find((item) => item.value === component)?.label ?? component;
}

function maintenanceIntervalLabel(task: GearMaintenanceTask): string {
  const parts = [];
  if (task.intervalDistance > 0) {
    parts.push(`${formatDistance(task.intervalDistance)}`);
  }
  if (task.intervalMonths > 0) {
    parts.push(`${task.intervalMonths} mo`);
  }
  return parts.length > 0 ? parts.join(" / ") : "-";
}

function maintenanceProgressLabel(task: GearMaintenanceTask): string {
  if (!task.lastMaintenance) return "No local record yet";
  if (task.intervalDistance > 0) {
    return `${formatDistance(task.distanceSince)} since service`;
  }
  if (task.intervalMonths > 0) {
    return `${task.monthsSince} months since service`;
  }
  return `Last done ${formatDate(task.lastMaintenance.date)}`;
}

function maintenanceHistoryLabel(record: GearMaintenanceRecord): string {
  return `${formatDate(record.date)} · ${formatDistance(record.distance)}`;
}

function isMaintenanceExpanded(gearId: string): boolean {
  return expandedMaintenanceByGearId[gearId] ?? false;
}

function toggleMaintenance(item: GearAnalysisItem) {
  expandedMaintenanceByGearId[item.id] = !isMaintenanceExpanded(item.id);
}

function todayInputValue(): string {
  return new Date().toISOString().slice(0, 10);
}

function openMaintenanceForm(item: GearAnalysisItem, task?: GearMaintenanceTask) {
  expandedMaintenanceByGearId[item.id] = true;
  maintenanceFormGearId.value = item.id;
  maintenanceForm.component = task?.componentLabel ?? "Chain";
  maintenanceForm.operation = task ? `${task.componentLabel} serviced` : "";
  maintenanceForm.date = todayInputValue();
  maintenanceForm.distanceKm = (item.distance / 1000).toFixed(0);
  maintenanceForm.note = "";
}

function closeMaintenanceForm() {
  maintenanceFormGearId.value = null;
}

function buildMaintenanceRequest(item: GearAnalysisItem): GearMaintenanceRecordRequest {
  const distanceKm = Number.parseFloat(maintenanceForm.distanceKm);
  return {
    gearId: item.id,
    component: maintenanceForm.component,
    operation: maintenanceForm.operation.trim() || `${maintenanceComponentLabel(maintenanceForm.component)} serviced`,
    date: maintenanceForm.date,
    distance: Number.isFinite(distanceKm) ? distanceKm * 1000 : item.distance,
    note: maintenanceForm.note.trim() || null,
  };
}

async function saveMaintenanceRecord(item: GearAnalysisItem) {
  savingMaintenanceGearId.value = item.id;
  try {
    await gearAnalysisStore.saveMaintenanceRecord(buildMaintenanceRequest(item));
    uiStore.showToast({
      id: `gear-maintenance-save-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: "Maintenance saved.",
      timeout: 2600,
    });
    closeMaintenanceForm();
  } catch (error) {
    uiStore.showToast({
      id: `gear-maintenance-save-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to save maintenance.",
      timeout: 3600,
    });
  } finally {
    savingMaintenanceGearId.value = null;
  }
}

async function markMaintenanceDone(item: GearAnalysisItem, task: GearMaintenanceTask) {
  savingMaintenanceGearId.value = item.id;
  try {
    await gearAnalysisStore.saveMaintenanceRecord({
      gearId: item.id,
      component: task.component,
      operation: `${task.componentLabel} serviced`,
      date: todayInputValue(),
      distance: item.distance,
      note: null,
    });
    uiStore.showToast({
      id: `gear-maintenance-done-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: "Maintenance marked as done.",
      timeout: 2600,
    });
  } catch (error) {
    uiStore.showToast({
      id: `gear-maintenance-done-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to update maintenance.",
      timeout: 3600,
    });
  } finally {
    savingMaintenanceGearId.value = null;
  }
}

async function deleteMaintenanceRecord(record: GearMaintenanceRecord) {
  deletingMaintenanceRecordId.value = record.id;
  try {
    await gearAnalysisStore.deleteMaintenanceRecord(record.id);
    uiStore.showToast({
      id: `gear-maintenance-delete-${Date.now()}`,
      type: ToastTypeEnum.NORMAL,
      message: "Maintenance removed.",
      timeout: 2600,
    });
  } catch (error) {
    uiStore.showToast({
      id: `gear-maintenance-delete-failed-${Date.now()}`,
      type: ToastTypeEnum.WARN,
      message: error instanceof Error ? error.message : "Unable to remove maintenance.",
      timeout: 3600,
    });
  } finally {
    deletingMaintenanceRecordId.value = null;
  }
}

function monthlyWidth(point: GearAnalysisPeriodPoint, points: GearAnalysisPeriodPoint[]): string {
  const max = Math.max(...points.map((entry) => entry.value), 1);
  return `${Math.max(4, Math.round((point.value / max) * 100))}%`;
}

function recentMonthly(points: GearAnalysisPeriodPoint[]): GearAnalysisPeriodPoint[] {
  return points.slice(-12);
}

function displayedMonthly(item: GearAnalysisItem): GearAnalysisPeriodPoint[] {
  return recentMonthly(item.monthlyDistance);
}

function monthlyCaption(item: GearAnalysisItem): string {
  const displayedMonths = displayedMonthly(item).length;
  if (currentYear.value === "All years") {
    return `Last ${displayedMonths} active months`;
  }
  return `${currentYear.value} active months`;
}

function monthLabel(point: GearAnalysisPeriodPoint): string {
  if (currentYear.value !== "All years") {
    return point.periodKey.slice(5);
  }
  const [year, month] = point.periodKey.split("-");
  return `${month}/${year.slice(2)}`;
}

function monthlyDistanceLabel(point: GearAnalysisPeriodPoint): string {
  return `${(point.value / 1000).toFixed(0)} km`;
}

function monthlyPointTitle(point: GearAnalysisPeriodPoint): string {
  return `${point.periodKey}: ${monthlyDistanceLabel(point)} over ${point.activityCount} activities`;
}
</script>

<template>
  <div class="gear-page">
    <section class="gear-summary">
      <div class="summary-tile">
        <span class="summary-label">Coverage</span>
        <strong>{{ coveragePercent }}%</strong>
        <span class="summary-detail">
          {{ analysis.coverage.assignedActivities }} / {{ analysis.coverage.totalActivities }}
        </span>
      </div>
      <div class="summary-tile">
        <span class="summary-label">Assigned</span>
        <strong>{{ formatDistance(analysis.items.reduce((sum, item) => sum + item.distance, 0)) }}</strong>
        <span class="summary-detail">{{ analysis.items.length }} gear</span>
      </div>
      <div class="summary-tile">
        <span class="summary-label">Unassigned</span>
        <strong>{{ formatDistance(analysis.unassigned.distance) }}</strong>
        <span class="summary-detail">{{ analysis.unassigned.activities }} activities</span>
      </div>
      <div class="summary-tile">
        <span class="summary-label">Moving time</span>
        <strong>{{ formatMovingTime(analysis.items.reduce((sum, item) => sum + item.movingTime, 0)) }}</strong>
        <span class="summary-detail">{{ formatElevation(analysis.items.reduce((sum, item) => sum + item.elevationGain, 0)) }}</span>
      </div>
      <div class="summary-tile">
        <span class="summary-label">Maintenance</span>
        <strong>{{ maintenanceDueCount }}</strong>
        <span class="summary-detail">due items</span>
      </div>
    </section>

    <section class="gear-toolbar">
      <label class="toolbar-field">
        <span>Type</span>
        <select v-model="kindFilter" class="form-select form-select-sm">
          <option value="ALL">All</option>
          <option value="BIKE">Bikes</option>
          <option value="SHOE">Shoes</option>
          <option value="UNKNOWN">Unknown</option>
          <option value="RETIRED">Retired</option>
        </select>
      </label>
      <label class="toolbar-field">
        <span>Sort</span>
        <select v-model="sortMode" class="form-select form-select-sm">
          <option value="distance">Distance</option>
          <option value="lastUsed">Last used</option>
          <option value="elevationGain">Elevation</option>
          <option value="activities">Activities</option>
        </select>
      </label>
    </section>

    <div v-if="isLoading" class="gear-state">
      Loading gear analysis...
    </div>
    <div v-else-if="error && !analysis.items.length" class="gear-state gear-state--error">
      {{ error }}
    </div>
    <template v-else>
      <div v-if="error" class="gear-state gear-state--warning">
        Displaying cached gear analysis. {{ error }}
      </div>

      <section v-if="filteredItems.length > 0" class="gear-list">
        <article
          v-for="item in filteredItems"
          :key="item.id"
          class="gear-row"
        >
          <div class="gear-main">
            <div class="gear-title-line">
              <h3>{{ item.name }}</h3>
              <span class="gear-pill">{{ gearKindLabel(item.kind) }}</span>
              <span v-if="item.primary" class="gear-pill gear-pill--primary">Primary</span>
              <span v-if="item.retired" class="gear-pill gear-pill--retired">Retired</span>
              <span :class="maintenanceClass(item)">{{ item.maintenanceLabel }}</span>
            </div>
            <div class="gear-dates">
              {{ formatDate(item.firstUsed) }} - {{ formatDate(item.lastUsed) }}
            </div>
          </div>

          <div class="gear-metrics">
            <div>
              <span>Distance</span>
              <strong>{{ formatDistance(item.distance) }}</strong>
            </div>
            <div>
              <span>Time</span>
              <strong>{{ formatMovingTime(item.movingTime) }}</strong>
            </div>
            <div>
              <span>D+</span>
              <strong>{{ formatElevation(item.elevationGain) }}</strong>
            </div>
            <div>
              <span>Avg</span>
              <strong>{{ formatSpeed(item) }}</strong>
            </div>
            <div>
              <span>Activities</span>
              <strong>{{ item.activities }}</strong>
            </div>
          </div>

          <div class="gear-best">
            <RouterLink
              v-if="item.longestActivity"
              :to="`/activities/${item.longestActivity.id}`"
            >
              Longest: {{ item.longestActivity.name }}
            </RouterLink>
            <RouterLink
              v-if="item.biggestElevationActivity"
              :to="`/activities/${item.biggestElevationActivity.id}`"
            >
              D+: {{ item.biggestElevationActivity.name }}
            </RouterLink>
            <RouterLink
              v-if="item.fastestActivity"
              :to="`/activities/${item.fastestActivity.id}`"
            >
              Fastest: {{ item.fastestActivity.name }}
            </RouterLink>
          </div>

          <div v-if="item.kind === 'BIKE'" class="maintenance-panel">
            <div class="maintenance-heading">
              <div>
                <span>Maintenance</span>
                <strong>{{ item.maintenanceHistory.length }} local records</strong>
              </div>
              <div class="maintenance-heading__actions">
                <button
                  type="button"
                  class="btn btn-sm btn-outline-primary"
                  :disabled="savingMaintenanceGearId === item.id"
                  @click="openMaintenanceForm(item)"
                >
                  <i class="fa-solid fa-screwdriver-wrench" aria-hidden="true" />
                  Add service record
                </button>
                <button
                  type="button"
                  class="btn btn-sm btn-outline-secondary"
                  @click="toggleMaintenance(item)"
                >
                  <i
                    :class="isMaintenanceExpanded(item.id) ? 'fa-solid fa-chevron-up' : 'fa-solid fa-chevron-down'"
                    aria-hidden="true"
                  />
                  {{ isMaintenanceExpanded(item.id) ? "Hide" : "Show" }}
                </button>
              </div>
            </div>

            <div v-if="isMaintenanceExpanded(item.id)" class="maintenance-body">
              <div v-if="item.maintenanceTasks.length" class="maintenance-task-list">
                <div
                  v-for="task in item.maintenanceTasks"
                  :key="`${item.id}-${task.component}`"
                  class="maintenance-task"
                >
                  <span :class="maintenanceTaskClass(task)">{{ task.status }}</span>
                  <div>
                    <strong>{{ task.componentLabel }}</strong>
                    <small>{{ task.statusLabel }} · {{ maintenanceProgressLabel(task) }} · interval {{ maintenanceIntervalLabel(task) }}</small>
                  </div>
                  <button
                    type="button"
                    class="btn btn-sm btn-outline-primary"
                    :disabled="savingMaintenanceGearId === item.id"
                    @click="markMaintenanceDone(item, task)"
                  >
                    <i class="fa-solid fa-check" aria-hidden="true" />
                    Mark as done
                  </button>
                </div>
              </div>

              <div class="maintenance-body-toolbar">
                <div>
                  <strong>Local service log</strong>
                  <small>Free-form component, date, odometer and note for this bike.</small>
                </div>
              </div>

              <form
                v-if="maintenanceFormGearId === item.id"
                class="maintenance-form"
                @submit.prevent="saveMaintenanceRecord(item)"
              >
                <label>
                  <span>Component</span>
                  <input
                    v-model="maintenanceForm.component"
                    class="form-control form-control-sm"
                    list="maintenance-component-suggestions"
                    type="text"
                    placeholder="Valve core rear"
                    required
                  >
                  <datalist id="maintenance-component-suggestions">
                    <option
                      v-for="component in maintenanceComponents"
                      :key="component.value"
                      :value="component.label"
                    />
                  </datalist>
                </label>
                <label>
                  <span>Operation</span>
                  <input
                    v-model="maintenanceForm.operation"
                    class="form-control form-control-sm"
                    type="text"
                    placeholder="Valve core changed"
                  >
                </label>
                <label>
                  <span>Date</span>
                  <input
                    v-model="maintenanceForm.date"
                    class="form-control form-control-sm"
                    type="date"
                    required
                  >
                </label>
                <label>
                  <span>Odometer (km)</span>
                  <input
                    v-model="maintenanceForm.distanceKm"
                    class="form-control form-control-sm"
                    type="number"
                    min="0"
                    step="1"
                    required
                  >
                </label>
                <label class="maintenance-form__note">
                  <span>Note</span>
                  <input
                    v-model="maintenanceForm.note"
                    class="form-control form-control-sm"
                    type="text"
                    placeholder="Optional"
                  >
                </label>
                <div class="maintenance-form__actions">
                  <button
                    type="submit"
                    class="btn btn-sm btn-primary"
                    :disabled="savingMaintenanceGearId === item.id"
                  >
                    Save
                  </button>
                  <button
                    type="button"
                    class="btn btn-sm btn-outline-secondary"
                    @click="closeMaintenanceForm"
                  >
                    Cancel
                  </button>
                </div>
              </form>

              <div v-if="item.maintenanceHistory.length" class="maintenance-history">
                <div
                  v-for="record in item.maintenanceHistory"
                  :key="record.id"
                  class="maintenance-history-row"
                >
                  <div>
                    <strong>{{ record.operation }}</strong>
                    <small>{{ record.componentLabel }} · {{ maintenanceHistoryLabel(record) }}</small>
                    <small v-if="record.note">{{ record.note }}</small>
                  </div>
                  <button
                    type="button"
                    class="btn btn-sm btn-outline-secondary"
                    :disabled="deletingMaintenanceRecordId === record.id"
                    @click="deleteMaintenanceRecord(record)"
                  >
                    <i class="fa-solid fa-trash" aria-hidden="true" />
                  </button>
                </div>
              </div>
              <div v-else class="maintenance-empty">
                No local service record yet.
              </div>
            </div>
          </div>

          <div v-if="item.monthlyDistance.length" class="monthly-panel">
            <div class="monthly-heading">
              <span>Monthly distance</span>
              <strong>{{ monthlyCaption(item) }}</strong>
            </div>
            <div class="monthly-strip">
              <div
                v-for="point in displayedMonthly(item)"
                :key="`${item.id}-${point.periodKey}`"
                class="month-bar"
                :title="monthlyPointTitle(point)"
              >
                <span>{{ monthLabel(point) }}</span>
                <div>
                  <i :style="{ width: monthlyWidth(point, displayedMonthly(item)) }" />
                </div>
                <strong>{{ monthlyDistanceLabel(point) }}</strong>
              </div>
            </div>
          </div>
        </article>
      </section>

      <div v-else class="gear-state">
        No assigned gear for the current filters.
      </div>

      <section v-if="analysis.unassigned.activities > 0" class="unassigned-band">
        <div>
          <h3>Unassigned</h3>
          <p>{{ analysis.unassigned.activities }} activities</p>
        </div>
        <div class="unassigned-metrics">
          <strong>{{ formatDistance(analysis.unassigned.distance) }}</strong>
          <span>{{ formatMovingTime(analysis.unassigned.movingTime) }}</span>
          <span>{{ formatElevation(analysis.unassigned.elevationGain) }}</span>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.gear-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.gear-summary {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.summary-tile {
  border: 1px solid var(--ms-border);
  border-radius: 8px;
  background: #ffffff;
  padding: 12px;
  min-width: 0;
}

.summary-label,
.summary-detail,
.gear-dates,
.gear-metrics span,
.month-bar span,
.month-bar strong,
.unassigned-band p {
  color: var(--ms-text-muted);
  font-size: 0.78rem;
  font-weight: 700;
}

.summary-tile strong {
  display: block;
  color: var(--ms-text);
  font-size: 1.35rem;
  line-height: 1.2;
  margin-top: 4px;
}

.summary-detail {
  display: block;
  margin-top: 2px;
}

.gear-toolbar {
  align-items: end;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: flex-end;
}

.toolbar-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 150px;
}

.toolbar-field span {
  color: var(--ms-text-muted);
  font-size: 0.75rem;
  font-weight: 800;
  text-transform: uppercase;
}

.gear-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.gear-row {
  border: 1px solid var(--ms-border);
  border-radius: 8px;
  background: #ffffff;
  display: grid;
  grid-template-columns: minmax(220px, 1.15fr) minmax(420px, 1.75fr);
  gap: 12px 16px;
  padding: 12px;
}

.gear-main,
.gear-best {
  min-width: 0;
}

.gear-title-line {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.gear-title-line h3,
.unassigned-band h3 {
  color: var(--ms-text);
  font-size: 1rem;
  line-height: 1.2;
  margin: 0;
}

.gear-pill {
  border: 1px solid #cbd5e1;
  border-radius: 999px;
  color: #475569;
  font-size: 0.68rem;
  font-weight: 800;
  line-height: 1;
  padding: 4px 7px;
  text-transform: uppercase;
}

.gear-pill--primary {
  border-color: #b7d7c2;
  color: #2e6b3f;
}

.gear-pill--retired {
  border-color: #e3c0bd;
  color: #9f3a32;
}

.gear-pill--maintenance-ok {
  border-color: #dbe2ea;
  color: #64748b;
}

.gear-pill--maintenance-soon {
  border-color: #f2cf8b;
  color: #8a5b1d;
}

.gear-pill--maintenance-due,
.gear-pill--maintenance-overdue {
  border-color: #e2aca8;
  color: #9f2d2d;
}

.gear-dates {
  margin-top: 5px;
}

.gear-metrics {
  display: grid;
  grid-template-columns: repeat(5, minmax(78px, 1fr));
  gap: 8px;
}

.gear-metrics div {
  border-left: 2px solid #f3d8ca;
  min-width: 0;
  padding-left: 8px;
}

.gear-metrics strong {
  color: var(--ms-text);
  display: block;
  font-size: 0.95rem;
  line-height: 1.2;
  margin-top: 2px;
  overflow-wrap: anywhere;
}

.gear-best {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.gear-best a {
  color: #315d84;
  font-size: 0.82rem;
  font-weight: 700;
  overflow: hidden;
  text-decoration: none;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gear-best a:hover {
  text-decoration: underline;
}

.maintenance-panel {
  border-top: 1px solid #edf0f4;
  display: grid;
  gap: 10px;
  grid-column: 1 / -1;
  min-width: 0;
  padding-top: 10px;
}

.maintenance-heading,
.maintenance-task,
.maintenance-history-row,
.maintenance-form__actions {
  align-items: center;
  display: flex;
  gap: 10px;
}

.maintenance-heading {
  justify-content: space-between;
}

.maintenance-heading__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: flex-end;
}

.maintenance-heading span,
.maintenance-form label span {
  color: var(--ms-text-muted);
  display: block;
  font-size: 0.74rem;
  font-weight: 800;
  text-transform: uppercase;
}

.maintenance-heading strong {
  color: var(--ms-text);
  display: block;
  font-size: 0.9rem;
}

.maintenance-heading .btn,
.maintenance-body-toolbar .btn,
.maintenance-task .btn,
.maintenance-history-row .btn {
  align-items: center;
  display: inline-flex;
  gap: 6px;
  min-height: 30px;
  white-space: nowrap;
}

.maintenance-task-list,
.maintenance-body,
.maintenance-history {
  display: grid;
  gap: 1px;
}

.maintenance-body {
  gap: 10px;
}

.maintenance-body-toolbar {
  align-items: center;
  background: #ffffff;
  border: 1px solid #edf0f4;
  border-radius: 8px;
  display: flex;
  gap: 10px;
  justify-content: space-between;
  padding: 8px;
}

.maintenance-body-toolbar strong {
  color: var(--ms-text);
  display: block;
  font-size: 0.86rem;
}

.maintenance-body-toolbar small {
  color: var(--ms-text-muted);
  display: block;
  font-size: 0.76rem;
  font-weight: 700;
}

.maintenance-task,
.maintenance-history-row {
  background: #fafbfe;
  border: 1px solid #edf0f4;
  border-radius: 6px;
  justify-content: space-between;
  padding: 8px;
}

.maintenance-empty {
  border: 1px dashed #dfe6ef;
  border-radius: 6px;
  color: var(--ms-text-muted);
  font-size: 0.8rem;
  font-weight: 700;
  padding: 8px;
}

.maintenance-task > div,
.maintenance-history-row > div {
  flex: 1 1 auto;
  min-width: 0;
}

.maintenance-task strong,
.maintenance-history-row strong {
  color: var(--ms-text);
  display: block;
  font-size: 0.86rem;
}

.maintenance-task small,
.maintenance-history-row small {
  color: var(--ms-text-muted);
  display: block;
  font-size: 0.76rem;
  font-weight: 700;
  overflow-wrap: anywhere;
}

.maintenance-status {
  border: 1px solid #d7dee8;
  border-radius: 999px;
  color: #5c6675;
  flex: 0 0 auto;
  font-size: 0.68rem;
  font-weight: 900;
  padding: 4px 7px;
}

.maintenance-status--soon {
  border-color: #f2cf8b;
  color: #8a5b1d;
}

.maintenance-status--due,
.maintenance-status--overdue {
  border-color: #e2aca8;
  color: #9f2d2d;
}

.maintenance-form {
  align-items: end;
  background: #f7f9fc;
  border: 1px solid #dfe6ef;
  border-radius: 8px;
  display: grid;
  gap: 8px;
  grid-template-columns: minmax(150px, 1fr) minmax(180px, 1.2fr) minmax(130px, 0.8fr) minmax(110px, 0.7fr) minmax(160px, 1fr) auto;
  padding: 10px;
}

.maintenance-form label {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.monthly-panel {
  grid-column: 2;
  min-width: 0;
}

.monthly-heading {
  align-items: baseline;
  display: flex;
  gap: 8px;
  justify-content: space-between;
  margin-bottom: 5px;
}

.monthly-heading span,
.monthly-heading strong {
  color: var(--ms-text-muted);
  font-size: 0.76rem;
  font-weight: 800;
  text-transform: uppercase;
}

.monthly-heading strong {
  color: #4f5663;
  text-transform: none;
}

.monthly-strip {
  display: grid;
  grid-template-columns: repeat(12, minmax(24px, 1fr));
  gap: 5px;
}

.month-bar {
  min-width: 0;
}

.month-bar div {
  background: #eef2f6;
  border-radius: 999px;
  height: 7px;
  overflow: hidden;
}

.month-bar i {
  background: var(--ms-primary);
  border-radius: inherit;
  display: block;
  height: 100%;
}

.month-bar strong {
  display: block;
  margin-top: 2px;
}

.gear-state,
.unassigned-band {
  border: 1px dashed var(--ms-border);
  border-radius: 8px;
  background: #fafbfd;
  color: var(--ms-text-muted);
  padding: 14px;
}

.gear-state--error {
  border-color: #ef9a9a;
  background: #fff3f3;
  color: #9f2d2d;
}

.gear-state--warning {
  border-color: #ffd59f;
  background: #fff9ef;
  color: #8a5b1d;
}

.unassigned-band {
  align-items: center;
  border-style: solid;
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.unassigned-band p {
  margin: 2px 0 0;
}

.unassigned-metrics {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  justify-content: flex-end;
}

.unassigned-metrics strong,
.unassigned-metrics span {
  color: var(--ms-text);
  font-weight: 800;
}

@media (max-width: 1100px) {
  .gear-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .gear-row {
    grid-template-columns: 1fr;
  }

  .monthly-panel,
  .maintenance-panel {
    grid-column: 1;
  }

  .maintenance-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .gear-summary,
  .gear-metrics {
    grid-template-columns: 1fr;
  }

  .maintenance-heading,
  .maintenance-body-toolbar,
  .maintenance-task,
  .maintenance-history-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .maintenance-heading__actions {
    justify-content: flex-start;
  }

  .maintenance-form {
    grid-template-columns: 1fr;
  }

  .gear-toolbar {
    justify-content: stretch;
  }

  .toolbar-field {
    flex: 1 1 100%;
  }

  .monthly-strip {
    grid-template-columns: repeat(6, minmax(28px, 1fr));
  }

  .unassigned-band {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
