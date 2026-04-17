<script setup lang="ts">
import type {Activity} from "@/models/activity.model";

defineProps<{
  model: Activity;
}>();
</script>

<template>
  <div class="activity-name-cell">
    <RouterLink
        :to="`/activities/${model.id}`"
        class="activity-link"
    >
      {{ model.name }}
    </RouterLink>
  </div>
</template>

<style scoped>
.activity-name-cell {
  display: flex;
  align-items: center;
  min-height: 28px;
}

.activity-link {
  color: var(--ms-primary);
  text-decoration: underline;
}
</style>
