<script setup lang="ts">
import type { Statistics } from "@/models/statistics.model";

defineProps<{
  model: Statistics; 
}>();

</script>

<template>
  <div class="combined-cell">
    <a
      :href="model.activity?.link"
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
