<template>
  <div class="speed-cell">
    <span>{{ value }}</span>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from "vue";


defineProps<{
  value: string;
}>();


</script>

<style scoped>
.speed-cell {
  text-align: right;
}
</style>
