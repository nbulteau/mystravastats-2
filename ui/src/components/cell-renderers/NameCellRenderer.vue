<script setup lang="ts">
import type { Activity } from "@/models/activity.model";
import { defineProps } from "vue";

defineProps<{
  model: Activity;
}>();

import { eventBus } from "@/main";

function handleDetailledActivityClick(link: string) {
  eventBus.emit('detailledActivityClick', link);
}

</script>

<template>
  <div>
    <button
      type="button"
      class="btn btn-light"
      @click="handleDetailledActivityClick($props.model.link)"
    >
      <img
        src="@/assets/buttons/eye.png"
        alt="Info"
        width="16"
        height="16"
      >
    </button>
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
