<template>
  <div class="elapsed-time-cell">
    {{ formattedElapsedTime }}
  </div>
</template>

<script setup lang="ts">
import { computed, type ComputedRef } from "vue";

const props = defineProps<{
  value: number;
}>();

const formattedElapsedTime: ComputedRef<string> = computed(() => {
  const hours = Math.floor((props.value ?? 0) / 3600);
  const minutes = Math.floor(((props.value ?? 0) % 3600) / 60);
  const seconds = (props.value ?? 0) % 60;

  if (hours === 0) {
    return `${minutes}m ${seconds}s`; // Customize the formatting as needed
  }
  return `${hours}h ${minutes}m ${seconds}s`; // Customize the formatting as needed
});
</script>

<style scoped>
.elapsed-time-cell {
  text-align: right; 
}
</style>
