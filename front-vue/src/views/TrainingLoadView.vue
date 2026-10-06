<script setup lang="ts">
import { computed, onMounted } from "vue";
import { RouterLink, useRoute } from "vue-router";
import { useTrainingLoadStore } from "@/stores/training-load";
import { useContextStore } from "@/stores/context";
import { hours, largestMeasuredContribution, loadNumber, loadReasons, shiftWeek, volumeComparison } from "@/services/training-load-presentation";

const store = useTrainingLoadStore();
const context = useContextStore();
const route = useRoute();
const current = computed(() => store.report?.weeks[0]);
const previous = computed(() => store.report?.weeks[1]);
const rows = computed(() => store.report?.activities.filter(row => current.value && row.date >= current.value.summary.startDate && row.date <= current.value.summary.endDate) ?? []);
const largest = computed(() => largestMeasuredContribution(rows.value));
const scale = computed(() => Math.max(1,...(current.value?.days.map(day => day.measuredLoad ?? 0) ?? [])));
const intensityTotal = computed(() => current.value ? current.value.summary.aerobicSeconds+current.value.summary.thresholdSeconds+current.value.summary.highIntensitySeconds : 0);
onMounted(() => {
  if (typeof route.query.week === "string" && /^\d{4}-\d{2}-\d{2}$/.test(route.query.week)) store.week = route.query.week;
  context.updateCurrentView("training-load");
});
function pickWeek(event: Event) { void store.changeWeek((event.target as HTMLInputElement).value); }
</script>

<template>
  <main class="training-page">
    <header class="training-heading">
      <div><span class="eyebrow">PROGRESS · WEEKLY REVIEW</span><h1>Training load</h1><p>Your week's effort, with the evidence behind every number.</p></div>
      <div class="week-picker">
        <button class="btn btn-outline-secondary" aria-label="Previous week" @click="store.changeWeek(shiftWeek(store.week,-1))">←</button>
        <label>Week containing <input type="date" :value="store.week" @change="pickWeek" /></label>
        <button class="btn btn-outline-secondary" aria-label="Next week" @click="store.changeWeek(shiftWeek(store.week,1))">→</button>
        <button class="btn btn-outline-primary" @click="store.load()">Refresh</button>
      </div>
    </header>
    <p class="subtle">Uses the selected sports and the week above, independently of the year filter. Power load currently covers cycling; other sports still contribute to volume.</p>
    <p v-if="store.loading" role="status" class="notice">Calculating the weekly report…</p>
    <div v-else-if="store.error" role="alert" class="notice error">{{ store.error }} <button class="btn btn-outline-danger" @click="store.load()">Retry</button></div>
    <template v-else-if="current && store.report">
      <h2>{{ current.summary.startDate }} — {{ current.summary.endDate }}</h2>
      <section class="summary-grid" aria-label="Weekly summary">
        <article class="metric"><span>Measured power load</span><strong>{{ loadNumber(current.summary.measuredLoad) }}</strong><small>points · known contributions only</small></article>
        <article class="metric"><span>Activity coverage</span><strong>{{ current.summary.scoredCount }} / {{ current.summary.activityCount }}</strong><small>activities with a calculated load</small></article>
        <article class="metric"><span>Recorded volume</span><strong>{{ hours(current.summary.movingSeconds) }}</strong><small>{{ (current.summary.distanceMeters/1000).toFixed(1) }} km</small></article>
        <article class="metric"><span>Elevation gain</span><strong>{{ Math.round(current.summary.elevationMeters).toLocaleString() }} m</strong><small>all recorded activities in the selection</small></article>
      </section>
      <p v-if="current.summary.estimatedLoad !== null" class="notice">Estimated-power load: <strong>{{ loadNumber(current.summary.estimatedLoad) }} points</strong>, kept separate from measured-power load.</p>
      <p v-if="current.summary.activityCount === 0" class="notice">No activities recorded for this week and these sports. This does not establish that the week was a rest week.</p>
      <p v-else-if="current.summary.scoredCount < current.summary.activityCount" class="notice">{{ current.summary.activityCount-current.summary.scoredCount }} activities have unknown load. The known total is incomplete; see the reasons below. <RouterLink to="/settings">Review dated FTP settings</RouterLink>.</p>
      <p v-if="store.report.undatedActivities" class="notice">{{ store.report.undatedActivities }} activities in the selected sports have no usable date and could not be assigned to a week.</p>

      <div class="review-grid">
        <section class="panel"><h2>Daily contributions</h2>
          <div v-for="day in current.days" :key="day.startDate" class="day-row">
            <span>{{ day.startDate.slice(5) }}</span>
            <div class="bar-track" aria-hidden="true"><div class="bar" :style="{width: `${(day.measuredLoad ?? 0)/scale*100}%`}" /></div>
            <span>{{ day.measuredLoad === null ? (day.estimatedLoad !== null ? 'No measured load' : day.activityCount ? 'Unknown' : 'No activity') : `${loadNumber(day.measuredLoad)} pts` }}<small v-if="day.estimatedLoad !== null">+ {{ loadNumber(day.estimatedLoad) }} estimated</small><small v-if="day.scoredCount < day.activityCount">{{ day.activityCount-day.scoredCount }} unscored</small></span>
          </div>
          <p class="subtle">Bars show measured-power load. Empty days mean no recorded activity, not confirmed rest.</p>
        </section>
        <section class="panel"><h2>What explains this week?</h2>
          <ul class="explanations">
            <li v-if="previous">{{ volumeComparison(current.summary,previous.summary) }} <small>The reference is a full calendar week; the selected week may still be in progress.</small></li>
            <li v-if="largest">Largest measured contribution: <RouterLink :to="`/activities/${largest.activityId}`">{{ largest.name }}</RouterLink>, {{ loadNumber(largest.load) }} points ({{ hours(largest.coveredSeconds) }}, normalized power {{ Math.round(largest.normalizedPower!) }} W, FTP {{ largest.ftp }} W).</li>
            <li v-if="current.summary.best5MinuteActivityId !== null">Best measured 5-minute power this week: <RouterLink :to="`/activities/${current.summary.best5MinuteActivityId}`">{{ Math.round(current.summary.best5MinutePower!) }} W</RouterLink>. This is the week's best effort, not an all-time record.</li>
            <li v-else>No fully covered, measured 5-minute effort is available.</li>
          </ul>
          <h3>Intensity · measured power</h3>
          <p v-if="!intensityTotal" class="subtle">Intensity is unavailable without power and dated FTP.</p>
          <dl v-else class="intensity"><dt>Up to 90% FTP</dt><dd>{{ hours(current.summary.aerobicSeconds) }}</dd><dt>90–120% FTP</dt><dd>{{ hours(current.summary.thresholdSeconds) }}</dd><dt>Above 120% FTP</dt><dd>{{ hours(current.summary.highIntensitySeconds) }}</dd></dl>
          <p class="subtle">Covered intervals only; missing time is excluded.</p>
        </section>
      </div>

      <section class="panel"><h2>Activities behind the numbers</h2>
        <div class="table-scroll"><table><thead><tr><th>Activity</th><th>Load</th><th>Power source</th><th>Power coverage</th><th>FTP used</th><th>Explanation</th></tr></thead>
          <tbody><tr v-for="row in rows" :key="row.activityId"><td><RouterLink :to="`/activities/${row.activityId}`">{{ row.name }}</RouterLink><small>{{ row.date }} · {{ row.sport }}</small></td><td>{{ loadNumber(row.load) }}<small v-if="row.load !== null">{{ Math.round(row.normalizedPower!) }} W normalized · IF {{ row.intensityFactor?.toFixed(2) }}</small></td><td><span class="source" :class="row.source">{{ row.source }}</span></td><td>{{ Math.round(row.coveredSeconds) }} / {{ row.elapsedSeconds }} s<small>{{ Math.round(row.recordedSeconds) }} s in power recording</small></td><td>{{ row.ftp === null ? 'Unavailable' : `${row.ftp} W` }}<small v-if="row.ftpEffectiveFrom">since {{ row.ftpEffectiveFrom }}</small></td><td>{{ loadReasons[row.reason] ?? row.reason }}</td></tr></tbody>
        </table></div>
      </section>
      <section class="panel"><h2>Five-week perspective</h2><p class="subtle">Same selected sports, Monday to Sunday. Totals include only recorded activities; compare coverage alongside load.</p>
        <div class="table-scroll"><table><thead><tr><th>Week</th><th>Moving time</th><th>Elevation</th><th>Measured load</th><th>Estimated load</th><th>Scored activities</th><th>Best measured 5 min</th></tr></thead>
          <tbody><tr v-for="week in store.report.weeks" :key="week.summary.startDate"><td><button class="week-link" @click="store.changeWeek(week.summary.startDate)">{{ week.summary.startDate }}</button></td><td>{{ hours(week.summary.movingSeconds) }}</td><td>{{ Math.round(week.summary.elevationMeters) }} m</td><td>{{ loadNumber(week.summary.measuredLoad) }}</td><td>{{ loadNumber(week.summary.estimatedLoad) }}</td><td>{{ week.summary.scoredCount }} / {{ week.summary.activityCount }}</td><td><RouterLink v-if="week.summary.best5MinuteActivityId !== null" :to="`/activities/${week.summary.best5MinuteActivityId}`">{{ Math.round(week.summary.best5MinutePower!) }} W</RouterLink><span v-else>Unavailable</span></td></tr></tbody>
        </table></div>
      </section>
      <details class="panel"><summary>How this report is calculated</summary>
        <p>Load = covered hours × (normalized power ÷ FTP)² × 100. One covered hour at a normalized power equal to FTP gives 100 points.</p>
        <p>FTP comes from your latest dated setting valid on the activity date. Today's profile FTP is never applied retrospectively. Measured and estimated power stay separate.</p>
        <p>Measurements cover the interval until the next reading. Missing readings and intervals longer than {{ store.report.maxGapSeconds }} seconds are unobserved. The last measurement is not extended. A score requires at least 30 seconds, a complete power recording and coverage of the activity from its start to its end (up to one final second may be absent).</p>
        <p>Normalized power uses a continuous 30-second rolling average. Excluded activities are omitted. No heart-rate score is mixed into this power score. Form and fatigue trends will be added separately.</p>
      </details>
    </template>
  </main>
</template>

<style scoped>
.training-page{max-width:1320px;margin:0 auto;padding:28px 22px 60px;color:var(--bs-body-color)}
.training-heading{display:flex;justify-content:space-between;gap:24px;align-items:center;flex-wrap:wrap}.eyebrow{font-size:.72rem;font-weight:750;letter-spacing:.13em;color:#477566}h1{font-size:2.4rem;margin:7px 0}h2{font-size:1.18rem;margin:0 0 18px}h3{font-size:1rem;margin-top:22px}.training-heading p,.subtle{color:var(--bs-secondary-color);font-size:.88rem}.week-picker{display:flex;align-items:flex-end;gap:8px}.week-picker label{display:grid;font-size:.75rem;gap:4px}.week-picker input{padding:7px;border:1px solid var(--bs-border-color);border-radius:6px;background:var(--bs-body-bg);color:inherit}.summary-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:14px;margin:18px 0}.metric,.panel{border:1px solid var(--bs-border-color);background:var(--bs-body-bg);border-radius:12px;padding:22px}.metric{display:flex;flex-direction:column;gap:8px}.metric span{font-size:.84rem;color:var(--bs-secondary-color)}.metric strong{font-size:1.85rem;font-weight:650}.metric small{font-size:.73rem;color:var(--bs-secondary-color)}.notice{padding:14px 18px;border-left:3px solid #b18a3e;background:rgba(177,138,62,.09);border-radius:4px;font-size:.9rem}.error{border-color:#c44}.review-grid{display:grid;grid-template-columns:1fr 1fr;gap:18px}.panel{margin:18px 0}.review-grid .panel{margin:0 0 18px}.day-row{display:grid;grid-template-columns:48px 1fr 116px;align-items:center;gap:12px;margin:13px 0;font-size:.8rem}.bar-track{height:10px;background:var(--bs-tertiary-bg);border-radius:6px;overflow:hidden}.bar{height:100%;background:#428778;border-radius:6px}.explanations{padding-left:18px;font-size:.9rem}.explanations li{margin-bottom:14px}small{display:block;color:var(--bs-secondary-color);font-weight:400;font-size:.76rem;margin-top:4px}.intensity{display:grid;grid-template-columns:1fr auto;font-size:.86rem;gap:7px}.intensity dd{margin:0}.table-scroll{overflow:auto}table{width:100%;font-size:.85rem;white-space:normal}th{text-align:left;font-size:.73rem;font-weight:650;color:var(--bs-secondary-color);padding:10px;border-bottom:1px solid var(--bs-border-color)}td{padding:14px 10px;border-bottom:1px solid var(--bs-border-color);min-width:95px;vertical-align:top}.source{display:inline-block;border-radius:14px;padding:3px 9px;background:var(--bs-tertiary-bg);font-size:.73rem}.measured{color:#236956;background:#e3f1ec}.estimated{color:#855813;background:#f7ecd5}.week-link{border:0;background:none;color:var(--bs-link-color);padding:0;text-decoration:underline}summary{cursor:pointer;font-weight:600}details p{margin-top:14px;font-size:.88rem}
@media(max-width:850px){.summary-grid{grid-template-columns:repeat(2,1fr)}.review-grid{grid-template-columns:1fr}.training-page{padding:20px 12px}.week-picker{flex-wrap:wrap}.metric strong{font-size:1.5rem}}
</style>
