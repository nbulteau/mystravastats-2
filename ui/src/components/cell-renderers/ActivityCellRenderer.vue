<script setup lang="ts">
import type { Statistics } from "@/models/statistics.model";

const props = defineProps<{
  model: Statistics; 
}>();
</script>

<template>
  <div class="combined-cell">
    <a
      :href="`https://www.strava.com/activities/${model.activity?.id}`"
      target="_blank"
      class="activity-link"
    >{{ model.activity?.name }}</a>
  </div>
</template>

<style scoped>
.combined-cell {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.activity-link {
  color: blue;
  text-decoration: underline;
}
</style>
