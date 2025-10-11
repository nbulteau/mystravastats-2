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

const gradient = computed(() => props.value ? props.value.toFixed(1) + ' %' : 'Not available');
</script>

<style scoped>
.gradient-cell {
  text-align: right;
}
</style>