<template>
  <div class="gradient-cell">
    {{ bestGradient }} %
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  value: number; 
}>();

const bestGradient = computed(() =>props.value);
</script>

<style scoped>
.gradient-cell {
  text-align: right;
}
</style>