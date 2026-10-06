<script setup lang="ts">

import { Chart } from 'highcharts-vue';
import { computed, } from "vue";
import type { Options } from 'highcharts';
import type { DetailedActivity } from '@/models/activity.model';

import { PowerTimeline } from "@/services/power-timeline";

const props = defineProps<{
    activity: DetailedActivity;
}>();


const powerDistributionChartOptions = computed<Options>(() => {
    const watts = props.activity?.stream?.watts ?? [];
    const times = props.activity?.stream?.time ?? [];
    const timeline = new PowerTimeline(watts, times);
    const zones: { [key: number]: number } = {};
    for (let i = 0; i < times.length - 1; i++) {
        if (!timeline.intervalValid(i)) continue;
        const zone = Math.floor(watts[i]! / 25) * 25;
        zones[zone] = (zones[zone] ?? 0) + times[i + 1]! - times[i]!;
    }

    // Prepare data for Highcharts
    const seriesData = Object.entries(zones).map(([power, seconds]) => ({
        x: parseInt(power),
        y: seconds,
        percentage: ((seconds / Math.max(1, timeline.coveredSeconds)) * 100).toFixed(1),
    }));

    const formatTimeString = (seconds: number) => {
        const minutes = Math.floor(seconds / 60);
        const remainingSeconds = seconds % 60;
        return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
    };

    return {
        chart: {
            type: 'column',
            backgroundColor: 'transparent',
            spacing: [10, 10, 15, 10], // [top, right, bottom, left]
            height: 250, // Reduce overall height
            style: {
                fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
            }
        },
        title: {
            text: 'Power Distribution',
            style: { fontSize: '14px' },
            margin: 5
        },
        xAxis: {
            title: { text: 'Power (watts)', style: { fontSize: '12px' } },
            labels: {
                formatter: function () {
                    return `${Number(this.value)}-${Number(this.value) + 24}`;
                },
                style: { fontSize: '11px' }
            },
            plotLines: [{
                color: '#FF0000',
                dashStyle: 'Dash',
                value: props.activity?.weightedAverageWatts ?? 0,
                width: 2,
                zIndex: 5,
                label: {
                    text: `Weighted Avg: ${Math.round(props.activity?.weightedAverageWatts ?? 0)}W`,
                    align: 'left',
                    rotation: 0,
                    x: 10,
                    style: {
                        color: '#FF0000',
                        fontSize: '11px'
                    },
                    y: 15
                }
            }]
        },
        yAxis: {
            title: {
                text: 'Time',
                style: { fontSize: '12px' }
            },
            labels: {
                formatter: function () {
                    return formatTimeString(Number(this.value));
                },
                style: { fontSize: '11px' }
            }
        },
        legend: {
            enabled: false
        },
        tooltip: {
            formatter: function () {
                const xValue = true ? this.x : 0;
                return `<b>${xValue}-${xValue + 24}W</b><br/>
                Time: ${formatTimeString(this.y ?? 0)}<br/>
                ${(this as any).point.percentage}%`;
            },
            style: { fontSize: '11px' }
        },
        plotOptions: {
            column: {
                pointPadding: 0,
                groupPadding: 0,
                borderWidth: 0,
                shadow: false
            }
        },
        series: [{
            type: 'column',
            name: 'Time in Zone',
            data: seriesData,
            color: '#2E86C1'
        }],
        credits: {
            enabled: false
        }
    };
});
</script>

<template>
  <div id="power-distribution-chart">
    <Chart :options="powerDistributionChartOptions" />
  </div>
</template>
