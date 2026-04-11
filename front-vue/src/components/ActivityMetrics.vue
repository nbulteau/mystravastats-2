<script setup lang="ts">
import type { DetailedActivity } from '@/models/activity.model';
import { computed } from 'vue';
import { formatSpeedWithUnit, formatTime } from "@/utils/formatters";
import { useContextStore } from '@/stores/context.js';
import TooltipHint from "@/components/TooltipHint.vue";
import {
  computeHeartRateZoneDistribution,
  resolveHeartRateZoneSettings,
} from '@/utils/heart-rate-zones';
import { getMetricTooltip } from "@/utils/metric-tooltips";

const props = defineProps<{
  activity: DetailedActivity;
}>();

const contextStore = useContextStore();

const cadenceUnit = computed(() => {
  if (props.activity.type?.endsWith("Run")) return "spm";
  return "rpm";
});

const averageCadenceDisplay = computed(() => {
  const cadence = props.activity.averageCadence ?? 0;
  if (cadence <= 0) return null;

  if (props.activity.type?.endsWith("Run")) {
    return cadence * 2;
  }

  return cadence;
});

const resolvedHeartRateSettings = computed(() => {
  return (
    contextStore.heartRateZoneAnalysis?.resolvedSettings ??
    resolveHeartRateZoneSettings(
      contextStore.heartRateZoneSettings,
      Math.trunc(props.activity.maxHeartrate ?? 0) || null,
    )
  );
});

const activityHeartRateZones = computed(() => {
  const stream = props.activity.stream;
  if (!stream) return null;
  return computeHeartRateZoneDistribution(
    stream.heartrate ?? null,
    stream.time ?? null,
    resolvedHeartRateSettings.value ?? null,
  );
});
</script>

<template>
  <div id="activity-details-container">
    <div id="activity-details">
      <h3>Basic Information</h3>
      <ul>
        <li><strong>Type:</strong> {{ activity.type }}</li>
        <li>
          <strong>Date:</strong>
          {{
            activity.startDate
              ? new Date(activity.startDate).toLocaleDateString("en-GB", {
                weekday: "long",
                day: "numeric",
                month: "long",
                year: "numeric",
              })
              : "N/A"
          }}
        </li>
        <li>
          <strong>Distance:</strong>
          {{ ((activity.distance ?? 0) / 1000).toFixed(1) }} km
        </li>
        <li>
          <strong>Total Elevation Gain:</strong>
          {{ activity.totalElevationGain.toFixed(0) }} m
        </li>
        <li>
          <strong>Elapsed Time:</strong> {{ formatTime(activity?.elapsedTime ?? 0) }}
        </li>
        <li><strong>Moving Time:</strong> {{ formatTime(activity.movingTime ?? 0) }}</li>
      </ul>
    </div>
    <div id="performance-metrics">
      <h3>Performance Metrics</h3>
      <ul>
        <li>
          <strong>
            Average Speed:
            <TooltipHint :text="getMetricTooltip('Average Speed') ?? ''" />
          </strong>
          {{ formatSpeedWithUnit(activity.averageSpeed ?? 0, activity.type ?? "Ride") }}
        </li>
        <li>
          <strong>
            Max Speed:
            <TooltipHint :text="getMetricTooltip('Max Speed') ?? ''" />
          </strong>
          {{ formatSpeedWithUnit(activity.maxSpeed ?? 0, activity.type ?? "Ride") }}
        </li>
        <li v-if="averageCadenceDisplay !== null">
          <strong>
            Average Cadence:
            <TooltipHint :text="getMetricTooltip('Average Cadence') ?? ''" />
          </strong>
          {{ averageCadenceDisplay.toFixed(0) }} {{ cadenceUnit }}
        </li>
        <li v-if="(activity.averageWatts ?? 0) > 0">
          <strong>
            Average Watts:
            <TooltipHint :text="getMetricTooltip('Average Watts') ?? ''" />
          </strong>
          {{ (activity.averageWatts ?? 0).toFixed(0) }} W
        </li>
        <li v-if="(activity.weightedAverageWatts ?? 0) > 0">
          <strong>
            Weighted Average Watts:
            <TooltipHint :text="getMetricTooltip('Weighted Average Watts') ?? ''" />
          </strong>
          {{ (activity.weightedAverageWatts ?? 0).toFixed(0) }} W
        </li>
        <li v-if="(activity.kilojoules ?? 0) > 0">
          <strong>
            Kilojoules:
            <TooltipHint :text="getMetricTooltip('Kilojoules') ?? ''" />
          </strong>
          {{ (activity.kilojoules ?? 0).toFixed(0) }} kJ
        </li>
      </ul>
    </div>
    <div id="heart-rate-metrics">
      <h3>Heart Rate Metrics</h3>
      <ul>
        <li v-if="(activity.averageHeartrate ?? 0) > 0">
          <strong>
            Average Heartrate:
            <TooltipHint :text="getMetricTooltip('Average Heartrate') ?? ''" />
          </strong>
          {{ activity.averageHeartrate.toFixed(0) }} bpm
        </li>
        <li v-if="(activity.maxHeartrate ?? 0) > 0">
          <strong>
            Max Heartrate:
            <TooltipHint :text="getMetricTooltip('Max Heartrate') ?? ''" />
          </strong>
          {{ activity.maxHeartrate.toFixed(0) }} bpm
        </li>
        <li v-if="activityHeartRateZones?.easyHardRatio !== null && activityHeartRateZones?.easyHardRatio !== undefined">
          <strong>
            Easy/Hard Ratio:
            <TooltipHint :text="getMetricTooltip('Easy / Hard Ratio') ?? ''" />
          </strong>
          {{ activityHeartRateZones.easyHardRatio.toFixed(2) }} : 1
        </li>
        <li v-if="activityHeartRateZones">
          <strong>
            Tracked HR time:
            <TooltipHint :text="getMetricTooltip('Tracked HR time') ?? ''" />
          </strong>
          {{ formatTime(activityHeartRateZones.totalTrackedSeconds) }}
        </li>
        <li
          v-for="zone in activityHeartRateZones?.zones ?? []"
          :key="zone.zone"
        >
          <strong>{{ zone.zone }}:</strong>
          {{ formatTime(zone.seconds) }} ({{ zone.percentage.toFixed(1) }}%)
        </li>
        <li v-if="!activityHeartRateZones">
          <span class="text-muted">No heart rate stream available for zone analysis.</span>
        </li>
      </ul>
    </div>
  </div>
</template>

<style lang="scss" scoped>  
#activity-details-container {
  display: flex;
  justify-content: space-between;
  margin-top: 20px;
}

#activity-details,
#performance-metrics,
#heart-rate-metrics {
  width: 32%;
}
</style>
