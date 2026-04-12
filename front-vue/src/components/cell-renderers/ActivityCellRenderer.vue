<script setup lang="ts">
import type { ActivityShort } from "@/models/activity.model";

const props = defineProps<{
  model: {
    activity?: ActivityShort;
  };
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
  color: var(--ms-primary);
  text-decoration: underline;
}
</style>
