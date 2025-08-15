<template>
  <div class="gradient-cell">
    {{ bestGradient }}
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  value: string; 
}>();

const bestGradient = computed(() => props.value ? props.value + ' %' : 'Not available');
</script>

<style scoped>
.gradient-cell {
  text-align: right;
}
</style>