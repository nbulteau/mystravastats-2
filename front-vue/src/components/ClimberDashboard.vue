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
  { title: "VAM record", record: lifetime.value.records.vam, formatter: (value: number) => `${Math.round(value).toLocaleString("fr-FR")} m/h` },
  { title: "Plus long versant", record: lifetime.value.records.longest, formatter: (value: number) => `${value.toLocaleString("fr-FR", { maximumFractionDigits: 1 })} km` },
  { title: "Col le plus difficile", record: lifetime.value.records.hardest, formatter: (value: number) => `${Math.round(value).toLocaleString("fr-FR")} pts` },
  { title: "Col le plus gravi", record: lifetime.value.records.mostClimbed, formatter: (value: number) => `${Math.round(value)} ascension${value > 1 ? "s" : ""}` },
]);
const breakdowns = computed(() => [
  { title: "Par année", items: lifetime.value.years },
  { title: "Par pays", items: lifetime.value.countries },
  { title: "Par massif", items: lifetime.value.massifs },
  { title: "Par catégorie", items: lifetime.value.categories },
  { title: "Par altitude", items: lifetime.value.altitudeBands },
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
        <p>Vue grimpeur</p>
        <h2>Mes sommets, mes volumes et mes records</h2>
      </div>
      <span>Données calculées uniquement sur le même catalogue que Climb badges</span>
    </header>

    <div class="scope-grid">
      <article>
        <span>Période sélectionnée</span>
        <h3>{{ periodLabel }}</h3>
        <dl>
          <div><dt>Cols</dt><dd>{{ period.climbedSummits }}</dd></div>
          <div><dt>Versants</dt><dd>{{ period.climbedVariants }}</dd></div>
          <div><dt>Ascensions</dt><dd>{{ period.ascentCount }}</dd></div>
          <div><dt>D+ sur cols</dt><dd>{{ Math.round(period.climbElevationGain).toLocaleString("fr-FR") }} m</dd></div>
        </dl>
      </article>
      <article class="scope-lifetime">
        <span>Tous les ans</span>
        <h3>Carrière complète</h3>
        <dl>
          <div><dt>Cols</dt><dd>{{ lifetimeLoading ? "…" : lifetime.climbedSummits }}</dd></div>
          <div><dt>Ascensions</dt><dd>{{ lifetimeLoading ? "…" : lifetime.ascentCount }}</dd></div>
          <div><dt>Altitude de sommets cumulée</dt><dd>{{ lifetimeLoading ? "…" : `${Math.round(lifetime.cumulativeSummitAltitude).toLocaleString("fr-FR")} m` }}</dd></div>
          <div><dt>D+ sur cols</dt><dd>{{ lifetimeLoading ? "…" : `${Math.round(lifetime.climbElevationGain).toLocaleString("fr-FR")} m` }}</dd></div>
        </dl>
      </article>
    </div>
    <p v-if="lifetimeError" class="dashboard-warning">Les statistiques de carrière n’ont pas pu être chargées : {{ lifetimeError }}</p>

    <section aria-labelledby="climber-records-title">
      <div class="section-heading">
        <div><p>Records de carrière</p><h3 id="climber-records-title">Mes repères</h3></div>
        <span v-if="lifetime.excludedRecordCount">{{ lifetime.excludedRecordCount }} passage{{ lifetime.excludedRecordCount > 1 ? "s" : "" }} ignoré{{ lifetime.excludedRecordCount > 1 ? "s" : "" }} par les garde-fous qualité</span>
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
          <strong>{{ card.record ? card.formatter(card.record.value) : "Indisponible" }}</strong>
          <small>{{ card.record?.label ?? "Aucune donnée fiable" }}</small>
          <em v-if="card.record?.estimated">Alignement GPS estimé</em>
        </button>
      </div>
    </section>

    <section aria-labelledby="climber-progression-title">
      <div class="section-heading">
        <div><p>Progression de carrière</p><h3 id="climber-progression-title">Explorer mes ascensions</h3></div>
        <span>Chaque indicateur ouvre les cols qui composent son total</span>
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
            <strong>{{ item.ascentCount }} asc. · {{ item.climbCount }} versant{{ item.climbCount > 1 ? "s" : "" }}</strong>
          </button>
          <p v-if="!breakdown.items.length">Aucune ascension</p>
        </article>
      </div>
    </section>

    <p class="method-note">
      Le D+ utilise les caractéristiques cataloguées de chaque versant multipliées par le nombre de passages. La VAM utilise le temps détecté entre les points de départ et d’arrivée ; les flux incomplets ou dont la distance diffère de plus de 10 % sont exclus des records.
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
