<script setup lang="ts">
import type { DetailedActivity } from '@/models/activity.model';
import { onMounted } from 'vue';
import { formatSpeedWithUnit, formatTime } from "@/utils/formatters";

defineProps<{
  activity: DetailedActivity;
}>();

onMounted(() => {
  console.log('ActivityMetrics component mounted');
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
          <strong>Average Speed:</strong>
          {{ formatSpeedWithUnit(activity.averageSpeed ?? 0, activity.type ?? "Ride") }}
        </li>
        <li>
          <strong>Max Speed:</strong>
          {{ formatSpeedWithUnit(activity.maxSpeed ?? 0, activity.type ?? "Ride") }}
        </li>
        <li v-if="(activity.averageCadence ?? 0) > 0.0">
          <strong>Average Cadence:</strong>
          {{ ((activity.averageCadence ?? 0) * 2).toFixed(0) }} spm
        </li>
        <li v-if="(activity.averageWatts ?? 0) > 0">
          <strong>Average Watts:</strong> {{ (activity.averageWatts ?? 0).toFixed(0) }} W
        </li>
        <li v-if="(activity.weightedAverageWatts ?? 0) > 0">
          <strong>Weighted Average Watts:</strong>
          {{ (activity.weightedAverageWatts ?? 0).toFixed(0) }} W
        </li>
        <li v-if="(activity.kilojoules ?? 0) > 0">
          <strong>Kilojoules:</strong> {{ (activity.kilojoules ?? 0).toFixed(0) }} kJ
        </li>
      </ul>
    </div>
    <div id="heart-rate-metrics">
      <h3>Heart Rate Metrics</h3>
      <ul>
        <li v-if="(activity.averageHeartrate ?? 0) > 0">
          <strong>Average Heartrate:</strong>
          {{ activity.averageHeartrate.toFixed(0) }} bpm
        </li>
        <li v-if="(activity.maxHeartrate ?? 0) > 0">
          <strong>Max Heartrate:</strong> {{ activity.maxHeartrate.toFixed(0) }} bpm
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