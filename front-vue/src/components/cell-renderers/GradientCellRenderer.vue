<template>
  <div class="gradient-cell">
    {{ gradient }}
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  value: number;
}>();

const gradient = computed(() => {
  const numericValue = Number(props.value);
  if (!Number.isFinite(numericValue)) {
    return "Not available";
  }
  return `${numericValue.toFixed(1)} %`;
});
</script>

<style scoped>
.gradient-cell {
  text-align: right;
}
</style>
