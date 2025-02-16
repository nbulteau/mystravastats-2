<template>
  <div class="elevation-gain-cell">
    {{ elevationGain }} m
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  value: number; 
}>();

const elevationGain = computed(() =>props.value);
</script>

<style scoped>
.elevation-gain-cell {
  text-align: right;
}
</style>