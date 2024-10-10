<template>
  <div class="distance-cell">
    {{ formattedDistance }} km
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  value: number;
}>();

const formattedDistance = computed(() => {
  return (props.value / 1000).toFixed(1);
});
</script>

<style scoped>
.distance-cell {
  text-align: right; /* Aligns the text to the right */
}
</style>