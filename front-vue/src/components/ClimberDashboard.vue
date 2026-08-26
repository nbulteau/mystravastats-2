<script setup lang="ts">
import { computed } from "vue";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import {
  buildClimberDashboardStats,
  type ClimberBreakdownItem,
  type ClimberRecord,
} from "@/utils/climber-dashboard";

const props = defineProps<{
  periodClimbs: BadgeCheckResult[];
  lifetimeClimbs: BadgeCheckResult[];
  periodLabel: string;
  lifetimeLoading?: boolean;
  lifetimeError?: string;
}>();
const emit = defineEmits<{
  showClimbs: [variantIds: string[], label: string];
}>();

const period = computed(() => buildClimberDashboardStats(props.periodClimbs));
const lifetime = computed(() => buildClimberDashboardStats(props.lifetimeClimbs));
const recordCards = computed(() => [
  { title: "VAM record", record: lifetime.value.records.vam, formatter: (value: number) => `${Math.round(value).toLocaleString("en-US")} m/h` },
  { title: "Longest climb side", record: lifetime.value.records.longest, formatter: (value: number) => `${value.toLocaleString("en-US", { maximumFractionDigits: 1 })} km` },
  { title: "Hardest climb", record: lifetime.value.records.hardest, formatter: (value: number) => `${Math.round(value).toLocaleString("en-US")} pts` },
  { title: "Most climbed", record: lifetime.value.records.mostClimbed, formatter: (value: number) => `${Math.round(value)} ascent${value > 1 ? "s" : ""}` },
]);
const breakdowns = computed(() => [
  { title: "By year", items: lifetime.value.years },
  { title: "By country", items: lifetime.value.countries },
  { title: "By mountain range", items: lifetime.value.massifs },
  { title: "By category", items: lifetime.value.categories },
  { title: "By altitude", items: lifetime.value.altitudeBands },
]);

function showRecord(record: ClimberRecord | null, title: string): void {
  if (record) emit("showClimbs", record.variantIds, `${title} · ${record.label}`);
}

function showBreakdown(item: ClimberBreakdownItem, title: string): void {
  emit("showClimbs", item.variantIds, `${title} · ${item.label}`);
}
</script>

<template>
  <div class="climber-dashboard">
    <header class="dashboard-heading">
      <div>
        <p>Climber view</p>
        <h2>My summits, totals and records</h2>
      </div>
      <span>Calculated using the same catalogue as Climb badges</span>
    </header>

    <div class="scope-grid">
      <article>
        <span>Selected period</span>
        <h3>{{ periodLabel }}</h3>
        <dl>
          <div><dt>Climbs</dt><dd>{{ period.climbedSummits }}</dd></div>
          <div><dt>Sides</dt><dd>{{ period.climbedVariants }}</dd></div>
          <div><dt>Ascents</dt><dd>{{ period.ascentCount }}</dd></div>
          <div><dt>Climb elevation</dt><dd>{{ Math.round(period.climbElevationGain).toLocaleString("en-US") }} m</dd></div>
        </dl>
      </article>
      <article class="scope-lifetime">
        <span>All years</span>
        <h3>Lifetime</h3>
        <dl>
          <div><dt>Climbs</dt><dd>{{ lifetimeLoading ? "…" : lifetime.climbedSummits }}</dd></div>
          <div><dt>Ascents</dt><dd>{{ lifetimeLoading ? "…" : lifetime.ascentCount }}</dd></div>
          <div><dt>Cumulative summit altitude</dt><dd>{{ lifetimeLoading ? "…" : `${Math.round(lifetime.cumulativeSummitAltitude).toLocaleString("en-US")} m` }}</dd></div>
          <div><dt>Climb elevation</dt><dd>{{ lifetimeLoading ? "…" : `${Math.round(lifetime.climbElevationGain).toLocaleString("en-US")} m` }}</dd></div>
        </dl>
      </article>
    </div>
    <p v-if="lifetimeError" class="dashboard-warning">Lifetime statistics could not be loaded: {{ lifetimeError }}</p>

    <section aria-labelledby="climber-records-title">
      <div class="section-heading">
        <div><p>Lifetime records</p><h3 id="climber-records-title">My benchmarks</h3></div>
        <span v-if="lifetime.excludedRecordCount">{{ lifetime.excludedRecordCount }} ascent{{ lifetime.excludedRecordCount > 1 ? "s" : "" }} excluded by quality safeguards</span>
      </div>
      <div class="record-grid">
        <button
          v-for="card in recordCards"
          :key="card.title"
          type="button"
          :disabled="!card.record"
          @click="showRecord(card.record, card.title)"
        >
          <span>{{ card.title }}</span>
          <strong>{{ card.record ? card.formatter(card.record.value) : "Unavailable" }}</strong>
          <small>{{ card.record?.label ?? "No reliable data" }}</small>
          <em v-if="card.record?.estimated">Estimated GPS alignment</em>
        </button>
      </div>
    </section>

    <section aria-labelledby="climber-progression-title">
      <div class="section-heading">
        <div><p>Lifetime progression</p><h3 id="climber-progression-title">Explore my ascents</h3></div>
        <span>Each metric opens the climbs included in its total</span>
      </div>
      <div class="breakdown-grid">
        <article v-for="breakdown in breakdowns" :key="breakdown.title">
          <h4>{{ breakdown.title }}</h4>
          <button
            v-for="item in breakdown.items.slice(0, 8)"
            :key="item.key"
            type="button"
            @click="showBreakdown(item, breakdown.title)"
          >
            <span>{{ item.label }}</span>
            <strong>{{ item.ascentCount }} asc. · {{ item.climbCount }} side{{ item.climbCount > 1 ? "s" : "" }}</strong>
          </button>
          <p v-if="!breakdown.items.length">No ascents</p>
        </article>
      </div>
    </section>

    <p class="method-note">
      Elevation gain uses each climb side's catalogue characteristics multiplied by the number of ascents. VAM uses the time detected between the start and finish points; incomplete streams or distances differing by more than 10% are excluded from records.
    </p>
  </div>
</template>

<style scoped>
.climber-dashboard { display: flex; flex-direction: column; gap: 18px; }.dashboard-heading,.section-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 18px; }.dashboard-heading p,.section-heading p { margin: 0 0 4px; color: #176d50; font-size: .7rem; font-weight: 850; letter-spacing: .08em; text-transform: uppercase; }.dashboard-heading h2,.section-heading h3 { margin: 0; font-size: 1.12rem; }.dashboard-heading > span,.section-heading > span { max-width: 420px; color: var(--ms-text-muted); font-size: .68rem; text-align: right; }
.scope-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }.scope-grid article { padding: 15px; border: 1px solid #d9e2ec; border-radius: 14px; background: #f8fafc; }.scope-grid .scope-lifetime { border-color: #b9dccd; background: #f3fbf7; }.scope-grid article > span { color: var(--ms-text-muted); font-size: .65rem; font-weight: 800; text-transform: uppercase; }.scope-grid h3 { margin: 3px 0 12px; font-size: .95rem; }.scope-grid dl { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 6px; margin: 0; }.scope-grid dl div { min-width: 0; padding: 8px; border-radius: 8px; background: rgb(255 255 255 / 80%); }.scope-grid dt { color: var(--ms-text-muted); font-size: .58rem; text-transform: uppercase; }.scope-grid dd { margin: 4px 0 0; font-size: .88rem; font-weight: 850; }
.record-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 8px; margin-top: 10px; }.record-grid button { display: flex; min-height: 118px; flex-direction: column; align-items: flex-start; padding: 12px; border: 1px solid #dfe4e9; border-radius: 11px; color: var(--ms-text); background: white; text-align: left; }.record-grid button:not(:disabled):hover { border-color: #ef9f79; transform: translateY(-1px); }.record-grid button:disabled { opacity: .65; }.record-grid span { color: var(--ms-text-muted); font-size: .64rem; font-weight: 800; text-transform: uppercase; }.record-grid strong { margin-top: 9px; color: #a63a0a; font-size: 1.05rem; }.record-grid small { margin-top: 3px; font-size: .7rem; }.record-grid em { margin-top: auto; color: #8b6a16; font-size: .58rem; font-style: normal; }
.breakdown-grid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 8px; margin-top: 10px; }.breakdown-grid article { padding: 10px; border: 1px solid #e0e5ea; border-radius: 11px; background: #fafbfc; }.breakdown-grid h4 { margin: 0 0 7px; font-size: .76rem; }.breakdown-grid button { display: flex; width: 100%; justify-content: space-between; gap: 7px; padding: 6px 4px; border: 0; border-top: 1px solid #eceff2; color: var(--ms-text); background: transparent; text-align: left; }.breakdown-grid button:hover span { color: var(--ms-primary); text-decoration: underline; }.breakdown-grid button span { overflow: hidden; font-size: .68rem; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }.breakdown-grid button strong { flex: none; color: var(--ms-text-muted); font-size: .58rem; }.breakdown-grid article > p { color: var(--ms-text-muted); font-size: .68rem; }
.method-note,.dashboard-warning { margin: 0; padding: 10px; border: 1px dashed #cad6e2; border-radius: 9px; color: var(--ms-text-muted); background: #f7f9fb; font-size: .68rem; }.dashboard-warning { color: #8d341f; border-color: #e7b9ad; background: #fff7f4; }
@media (max-width: 1100px) { .record-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }.breakdown-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }.scope-grid dl { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 680px) { .dashboard-heading,.section-heading { flex-direction: column; }.dashboard-heading > span,.section-heading > span { text-align: left; }.scope-grid,.record-grid,.breakdown-grid { grid-template-columns: 1fr; } }
</style>
