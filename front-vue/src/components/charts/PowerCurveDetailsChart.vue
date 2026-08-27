<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { Chart } from 'highcharts-vue';
import type { Options } from 'highcharts';
import type { DetailedActivity } from '@/models/activity.model';
import { buildPowerCurve } from '@/services/activity-power-analysis';
import { formatTime } from '@/utils/formatters';

// Props
const props = defineProps<{
  activity: DetailedActivity;
}>();

// Reactive data
const chartOptions = ref<Options>({});



// Computed properties
const powerCurveData = computed(() => {
  return buildPowerCurve(props.activity.stream.watts || []);
});



// Chart options
onMounted(() => {
  chartOptions.value = {
    chart: {
      type: 'line'
    },
    title: {
      text: 'Power Curve'
    },
    xAxis: {
      title: {
        text: 'Time (seconds)'
      },
      type: 'linear'
    },
    yAxis: {
      title: {
        text: 'Power (watts)'
      }
    },
    legend: {
      enabled: false
    },
    tooltip: {
      formatter: function () {
        const yValue = typeof this.y === 'number' ? this.y : 0;
        return `<b>${yValue.toFixed(0)}W</b><br/>
                Duration: ${formatTime(Number(this.x) ?? 0)}<br/>`;
      },
      style: { fontSize: '11px' }
    },

    series: [{
      type: 'line',
      name: 'Power Curve',
      data: powerCurveData.value
    }]
  };
});
</script>

<template>
  <div id="power-curve-details">
    <Chart :options="chartOptions" />
  </div>
</template>

<style scoped>
/* Add any additional styling here */
</style>
