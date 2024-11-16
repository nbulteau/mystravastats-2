<script setup lang="ts">
import { reactive, watch } from "vue";
import { Chart } from "highcharts-vue";
import type { SeriesColumnOptions } from "highcharts";
import Highcharts from 'highcharts';
import HighchartsMore from 'highcharts/highcharts-more';

HighchartsMore(Highcharts);

const props = defineProps<{
    averageCadence: Array<Array<number>>
}>();

const chartOptions: Highcharts.Options = reactive({
    chart: {
        zooming: {
            type: 'x',
        },
        type: 'area'
    },
    title: {
        text: 'Average cadence (steps per minute)',
        align: 'left'
    },
    subtitle: {
        text: document.ontouchstart === undefined ?
            'Click and drag in the plot area to zoom in' :
            'Pinch the chart to zoom in',
        align: 'left'
    },
    xAxis: {
        type: 'datetime'
    },
    yAxis: {
        title: {
            text: 'steps per minute'
        }
    },
    legend: {
        enabled: false
    },
    plotOptions: {
        area: {
            marker: {
                radius: 2
            },
            lineWidth: 1,
            color: {
                linearGradient: {
                    x1: 0,
                    y1: 0,
                    x2: 0,
                    y2: 1
                },
                stops: [
                    [0, 'rgb(199, 113, 243)'],
                    [0.7, 'rgb(76, 175, 254)']
                ]
            },
            states: {
                hover: {
                    lineWidth: 1
                }
            },
            threshold: null
        }
    },

    series: [
        {
            type: 'area',
            name: 'Cadence',
            data: props.averageCadence
        },
    ] as Highcharts.SeriesOptionsType[]
});

// Watch for changes in distanceByWeeks and update the chart data
watch(() => props.averageCadence, (newData) => {
    if (chartOptions.series && chartOptions.series.length > 0) {
        (chartOptions.series[0] as SeriesColumnOptions).data = newData;
   }
}, { immediate: true }); // Immediate to handle initial data
</script>


<template>
  <Chart :options="chartOptions" />
</template>

<style scoped>
#container {
  width: 100%;
  height: 400px;
}
</style>
