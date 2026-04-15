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
    <RouterLink
      v-if="model.activity"
      :to="`/activities/${model.activity?.id}`"
      class="activity-link"
    >{{ model.activity?.name }}</RouterLink>
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
