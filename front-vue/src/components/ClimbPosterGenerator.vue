<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import { useAthleteStore } from "@/stores/athlete";
import {
  buildClimbPosterSvg,
  CLIMB_POSTER_DESIGNS,
  posterDesignMaxClimbs,
  type ClimbPosterDesign,
  type ClimbPosterEntry,
} from "@/utils/climb-poster";
import {
  orderClimbsForPoster,
  selectTopClimbsForPoster,
  type ClimbSelectionOrder,
} from "@/utils/climb-poster-selection";

const props = defineProps<{
  climbs: BadgeCheckResult[];
  yearLabel: string;
  isLoading?: boolean;
}>();

const athleteStore = useAthleteStore();
const dialogRef = ref<HTMLDialogElement | null>(null);
const previewRef = ref<HTMLElement | null>(null);
const selectedDesign = ref<ClimbPosterDesign>("altitude");
const selectedClimbLabels = ref<string[]>([]);
const climbSelectionOrder = ref<ClimbSelectionOrder>("hardest");
const generatedSvg = ref("");
const generationError = ref("");

const availableClimbs = computed(() => props.climbs.filter((result) => (
  result.nbCheckedActivities > 0 &&
  result.climbDetails != null &&
  result.climbDetails.ascentCount > 0
)));

const selectedDesignDefinition = computed(() => (
  CLIMB_POSTER_DESIGNS.find((design) => design.id === selectedDesign.value) ?? CLIMB_POSTER_DESIGNS[0]
));
const selectionLimit = computed(() => posterDesignMaxClimbs(selectedDesign.value));
const selectionPresetCount = computed(() => Math.min(selectionLimit.value, availableClimbs.value.length));
const displayedClimbs = computed(() => (
  orderClimbsForPoster(availableClimbs.value, climbSelectionOrder.value)
));
const selectedClimbs = computed(() => {
  const climbsByLabel = new Map(availableClimbs.value.map((result) => [result.badge.label, result]));
  return selectedClimbLabels.value
    .map((label) => climbsByLabel.get(label))
    .filter((result): result is BadgeCheckResult => result != null);
});

watch(selectedDesign, () => {
  selectedClimbLabels.value = selectedClimbLabels.value.slice(0, selectionLimit.value);
  invalidateGeneratedPoster();
});

watch(
  () => props.climbs,
  () => {
    const availableLabels = new Set(availableClimbs.value.map((result) => result.badge.label));
    selectedClimbLabels.value = selectedClimbLabels.value.filter((label) => availableLabels.has(label));
    invalidateGeneratedPoster();
  },
);

function openGenerator() {
  selectHardest();
  generatedSvg.value = "";
  generationError.value = "";
  dialogRef.value?.showModal();
}

function selectHardest() {
  selectBy("hardest");
}

function selectLongest() {
  selectBy("longest");
}

function selectBy(order: ClimbSelectionOrder) {
  climbSelectionOrder.value = order;
  selectedClimbLabels.value = selectTopClimbsForPoster(availableClimbs.value, order, selectionLimit.value)
    .map((result) => result.badge.label);
  invalidateGeneratedPoster();
}

function closeGenerator() {
  dialogRef.value?.close();
}

function closeOnBackdrop(event: MouseEvent) {
  if (event.target === dialogRef.value) {
    closeGenerator();
  }
}

function toggleClimb(label: string) {
  if (selectedClimbLabels.value.includes(label)) {
    selectedClimbLabels.value = selectedClimbLabels.value.filter((candidate) => candidate !== label);
  } else if (selectedClimbLabels.value.length < selectionLimit.value) {
    selectedClimbLabels.value = [...selectedClimbLabels.value, label];
  }
  invalidateGeneratedPoster();
}

function invalidateGeneratedPoster() {
  generatedSvg.value = "";
  generationError.value = "";
}

async function generatePoster() {
  generationError.value = "";
  try {
    const entries: ClimbPosterEntry[] = selectedClimbs.value.map((result) => ({
      label: result.badge.label,
      category: result.badge.category,
      details: result.climbDetails!,
    }));
    generatedSvg.value = buildClimbPosterSvg({
      design: selectedDesign.value,
      climbs: entries,
      yearLabel: props.yearLabel,
      athleteName: athleteStore.athleteName,
    });
    await nextTick();
    previewRef.value?.scrollIntoView({ behavior: "smooth", block: "start" });
  } catch (error) {
    generationError.value = error instanceof Error ? error.message : "Unable to generate the poster.";
  }
}

function downloadPoster() {
  if (!generatedSvg.value) {
    return;
  }
  const blob = new Blob([generatedSvg.value], { type: "image/svg+xml;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `climb-poster-${selectedDesign.value}-${slugify(props.yearLabel)}.svg`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function slugify(value: string): string {
  return value
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "") || "all-time";
}
</script>

<template>
  <div class="poster-generator-entry">
    <button
      type="button"
      class="btn btn-primary btn-sm"
      :disabled="props.isLoading || availableClimbs.length === 0"
      @click="openGenerator"
    >
      <i class="fa-solid fa-image" aria-hidden="true" />
      {{ props.isLoading ? "Updating climbs..." : "Create climb poster" }}
    </button>
    <span v-if="props.isLoading" class="poster-generator-hint">
      Updating climbed cols for {{ yearLabel }}.
    </span>
    <span v-else-if="availableClimbs.length === 0" class="poster-generator-hint">
      Unlock a famous climb to create a poster.
    </span>
  </div>

  <dialog
    ref="dialogRef"
    class="poster-dialog"
    aria-labelledby="poster-dialog-title"
    @click="closeOnBackdrop"
  >
    <div class="poster-dialog-shell">
      <header class="poster-dialog-header">
        <div>
          <p class="poster-dialog-kicker">Print studio</p>
          <h2 id="poster-dialog-title">Create a climb poster</h2>
          <p>Choose a design, select your cols, then generate a print-ready SVG.</p>
        </div>
        <button type="button" class="btn btn-light poster-dialog-close" aria-label="Close poster generator" @click="closeGenerator">
          <i class="fa-solid fa-xmark" aria-hidden="true" />
        </button>
      </header>

      <section class="poster-step" aria-labelledby="poster-design-heading">
        <div class="poster-step-heading">
          <span>1</span>
          <div>
            <h3 id="poster-design-heading">Choose a design</h3>
            <p>The design is applied when the poster is generated.</p>
          </div>
        </div>
        <div class="poster-design-grid">
          <label
            v-for="design in CLIMB_POSTER_DESIGNS"
            :key="design.id"
            class="poster-design-card"
            :class="{ 'poster-design-card--selected': selectedDesign === design.id }"
          >
            <input v-model="selectedDesign" type="radio" name="poster-design" :value="design.id">
            <span class="poster-design-miniature" :class="`poster-design-miniature--${design.id}`" aria-hidden="true">
              <span /><span /><span />
            </span>
            <strong>{{ design.name }}</strong>
            <small>{{ design.description }}</small>
            <span class="poster-design-capacity">Up to {{ design.maxClimbs }} cols</span>
          </label>
        </div>
      </section>

      <section class="poster-step" aria-labelledby="poster-climbs-heading">
        <div class="poster-step-heading poster-step-heading--split">
          <div class="poster-step-heading-main">
            <span>2</span>
            <div>
              <h3 id="poster-climbs-heading">Select climbed cols</h3>
              <p>{{ selectedClimbLabels.length }} of {{ selectionLimit }} selected from {{ availableClimbs.length }} climbed cols for {{ selectedDesignDefinition.name }}.</p>
            </div>
          </div>
          <div class="poster-selection-actions">
            <button
              type="button"
              class="btn btn-sm btn-outline-secondary"
              :class="{ active: climbSelectionOrder === 'hardest' }"
              :aria-pressed="climbSelectionOrder === 'hardest'"
              @click="selectHardest"
            >
              Select hardest {{ selectionPresetCount }}
            </button>
            <button
              type="button"
              class="btn btn-sm btn-outline-secondary"
              :class="{ active: climbSelectionOrder === 'longest' }"
              :aria-pressed="climbSelectionOrder === 'longest'"
              @click="selectLongest"
            >
              Select longest {{ selectionPresetCount }}
            </button>
          </div>
        </div>
        <div class="poster-climb-grid">
          <label
            v-for="result in displayedClimbs"
            :key="result.badge.label"
            class="poster-climb-choice"
            :class="{ 'poster-climb-choice--selected': selectedClimbLabels.includes(result.badge.label) }"
          >
            <input
              type="checkbox"
              :checked="selectedClimbLabels.includes(result.badge.label)"
              :disabled="!selectedClimbLabels.includes(result.badge.label) && selectedClimbLabels.length >= selectionLimit"
              @change="toggleClimb(result.badge.label)"
            >
            <span>
              <strong>{{ result.climbDetails?.name }}</strong>
              <small>
                {{ result.climbDetails?.country }} · {{ result.climbDetails?.massif }} ·
                {{ result.climbDetails?.lengthKm.toFixed(1) }} km ·
                {{ result.climbDetails?.averageGradient.toFixed(1) }}% ·
                difficulty {{ result.climbDetails?.difficulty }} ·
                {{ result.climbDetails?.ascentCount }} ascent{{ result.climbDetails?.ascentCount === 1 ? '' : 's' }}
              </small>
            </span>
          </label>
        </div>
      </section>

      <div class="poster-generate-bar">
        <div>
          <strong>{{ selectedClimbLabels.length }} cols · {{ selectedDesignDefinition.name }}</strong>
          <small>Up to 2000 × 3000 SVG, designed for a 60 × 90 cm poster</small>
        </div>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="props.isLoading || selectedClimbLabels.length === 0"
          @click="generatePoster"
        >
          <i class="fa-solid fa-wand-magic-sparkles" aria-hidden="true" />
          Generate poster
        </button>
      </div>

      <p v-if="generationError" class="poster-generation-error" role="alert">{{ generationError }}</p>

      <section v-if="generatedSvg" ref="previewRef" class="poster-result" aria-labelledby="poster-preview-heading">
        <div class="poster-result-header">
          <div>
            <h3 id="poster-preview-heading">Poster preview</h3>
            <p>The 50-col layout is optimised for 60 × 90 cm and stays sharp at any print size.</p>
          </div>
          <button type="button" class="btn btn-primary" @click="downloadPoster">
            <i class="fa-solid fa-download" aria-hidden="true" />
            Download SVG
          </button>
        </div>
        <div class="poster-preview" v-html="generatedSvg" />
      </section>
    </div>
  </dialog>
</template>

<style scoped>
.poster-generator-entry {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.poster-generator-entry .btn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  border-radius: 999px;
  font-weight: 700;
  padding-inline: 14px;
}

.poster-generator-hint {
  color: var(--ms-text-muted);
  font-size: 0.8rem;
  font-weight: 600;
}

.poster-dialog {
  width: min(1040px, calc(100vw - 28px));
  max-height: calc(100vh - 28px);
  padding: 0;
  border: 0;
  border-radius: 18px;
  background: var(--ms-surface-strong);
  color: var(--ms-text);
  box-shadow: 0 28px 80px rgba(20, 28, 42, 0.35);
}

.poster-dialog::backdrop {
  background: rgba(19, 25, 36, 0.68);
  backdrop-filter: blur(4px);
}

.poster-dialog-shell {
  padding: 22px;
}

.poster-dialog-header,
.poster-result-header,
.poster-generate-bar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 18px;
}

.poster-dialog-header {
  padding-bottom: 18px;
  border-bottom: 1px solid var(--ms-border);
}

.poster-dialog-header h2,
.poster-step h3,
.poster-result h3 {
  margin: 0;
  color: var(--ms-text);
  font-weight: 800;
}

.poster-dialog-header h2 {
  font-size: 1.45rem;
}

.poster-dialog-header p,
.poster-step-heading p,
.poster-result-header p {
  margin: 4px 0 0;
  color: var(--ms-text-muted);
  font-size: 0.88rem;
}

.poster-dialog-kicker {
  margin: 0 0 3px !important;
  color: var(--ms-primary) !important;
  font-size: 0.76rem !important;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.poster-dialog-close {
  flex: 0 0 auto;
  border-radius: 50%;
  width: 38px;
  height: 38px;
}

.poster-step {
  padding: 20px 0;
  border-bottom: 1px solid var(--ms-border);
}

.poster-step-heading,
.poster-step-heading-main {
  display: flex;
  align-items: flex-start;
  gap: 11px;
}

.poster-step-heading > span,
.poster-step-heading-main > span {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  border-radius: 50%;
  color: #fff;
  background: var(--ms-primary);
  font-size: 0.82rem;
  font-weight: 800;
}

.poster-step-heading h3,
.poster-result h3 {
  font-size: 1rem;
}

.poster-step-heading--split {
  justify-content: space-between;
  align-items: center;
}

.poster-selection-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.poster-design-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-top: 14px;
}

.poster-design-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  padding: 12px;
  background: var(--ms-surface);
  cursor: pointer;
  transition: border-color 0.15s ease, box-shadow 0.15s ease, transform 0.15s ease;
}

.poster-design-card:hover {
  transform: translateY(-2px);
  border-color: rgba(252, 76, 2, 0.5);
}

.poster-design-card--selected {
  border-color: var(--ms-primary);
  box-shadow: 0 0 0 2px rgba(252, 76, 2, 0.15);
}

.poster-design-card input {
  position: absolute;
  top: 10px;
  right: 10px;
  accent-color: var(--ms-primary);
}

.poster-design-card strong {
  color: var(--ms-text);
  font-size: 0.95rem;
}

.poster-design-card small {
  color: var(--ms-text-muted);
  line-height: 1.3;
}

.poster-design-capacity {
  margin-top: 2px;
  color: var(--ms-primary);
  font-size: 0.72rem;
  font-weight: 800;
  text-transform: uppercase;
}

.poster-design-miniature {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 6px;
  width: 100%;
  height: 74px;
  margin-bottom: 4px;
  padding: 12px;
  border-radius: 9px;
  background: #f4f0e8;
  overflow: hidden;
}

.poster-design-miniature span {
  display: block;
  height: 4px;
  border-radius: 99px;
  background: #d85f2f;
}

.poster-design-miniature span:nth-child(2) { width: 78%; }
.poster-design-miniature span:nth-child(3) { width: 56%; }
.poster-design-miniature--altitude {
  justify-content: flex-end;
  gap: 8px;
  background: #fbf8f1;
}
.poster-design-miniature--altitude span {
  height: 5px;
  background: #d65d2d;
  transform: rotate(-5deg);
  transform-origin: left center;
}
.poster-design-miniature--altitude span:nth-child(2) { width: 88%; }
.poster-design-miniature--altitude span:nth-child(3) { width: 70%; }
.poster-design-miniature--topo {
  background-color: #edf0eb;
  background-image: linear-gradient(rgba(30, 40, 50, 0.08) 1px, transparent 1px), linear-gradient(90deg, rgba(30, 40, 50, 0.08) 1px, transparent 1px);
  background-size: 12px 12px;
}
.poster-design-miniature--topo span {
  height: 3px;
  border-radius: 0;
  background: #19303a;
}
.poster-design-miniature--topo span:first-child {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: #d34f2f;
}
.poster-design-miniature--collection {
  flex-direction: row;
  align-items: flex-end;
  gap: 8px;
  background: #efe7d9;
}
.poster-design-miniature--collection span {
  width: 24%;
  height: 70%;
  border: 1px solid #c8bca9;
  border-top: 4px solid #b74d2d;
  border-radius: 5px;
  background: #fbf8f1;
}
.poster-design-miniature--collection span:nth-child(2) { width: 24%; height: 92%; }
.poster-design-miniature--collection span:nth-child(3) { width: 24%; height: 58%; }

.poster-climb-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 9px;
  max-height: 260px;
  overflow-y: auto;
  margin-top: 14px;
  padding-right: 4px;
}

.poster-climb-choice {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid var(--ms-border);
  border-radius: 11px;
  background: var(--ms-surface);
  cursor: pointer;
}

.poster-climb-choice--selected {
  border-color: rgba(252, 76, 2, 0.55);
  background: rgba(252, 76, 2, 0.06);
}

.poster-climb-choice:has(input:disabled) {
  opacity: 0.52;
  cursor: not-allowed;
}

.poster-climb-choice input {
  accent-color: var(--ms-primary);
}

.poster-climb-choice span {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.poster-climb-choice strong {
  color: var(--ms-text);
  font-size: 0.88rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.poster-climb-choice small {
  color: var(--ms-text-muted);
  font-size: 0.75rem;
}

.poster-generate-bar {
  align-items: center;
  margin-top: 20px;
  padding: 14px;
  border-radius: 13px;
  background: var(--ms-surface);
  border: 1px solid var(--ms-border);
}

.poster-generate-bar > div {
  display: flex;
  flex-direction: column;
}

.poster-generate-bar strong {
  color: var(--ms-text);
  font-size: 0.92rem;
}

.poster-generate-bar small {
  color: var(--ms-text-muted);
  font-size: 0.76rem;
}

.poster-generation-error {
  margin: 12px 0 0;
  color: #a43116;
  font-weight: 700;
}

.poster-result {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--ms-border);
}

.poster-result-header {
  align-items: center;
  margin-bottom: 14px;
}

.poster-preview {
  width: min(100%, 620px);
  margin: 0 auto;
  padding: 10px;
  border-radius: 12px;
  background: #dfe3e8;
  box-shadow: inset 0 0 0 1px rgba(25, 35, 50, 0.1);
}

.poster-preview :deep(svg) {
  display: block;
  width: 100%;
  height: auto;
  box-shadow: 0 12px 30px rgba(24, 32, 44, 0.2);
}

@media (max-width: 760px) {
  .poster-dialog-shell { padding: 16px; }
  .poster-design-grid { grid-template-columns: 1fr; }
  .poster-climb-grid { grid-template-columns: 1fr; }
  .poster-step-heading--split,
  .poster-generate-bar,
  .poster-result-header { align-items: stretch; flex-direction: column; }
  .poster-generate-bar .btn,
  .poster-result-header .btn { width: 100%; }
}
</style>
