<script setup lang="ts">
import type { Activity } from "@/models/activity.model";

defineProps<{
  model: Activity; 
}>();

</script>

<template>
  <div class="combined-cell">
    <a
      :href="model.link"
      target="_blank"
      class="activity-link"
    >{{ model.name }}</a>
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
