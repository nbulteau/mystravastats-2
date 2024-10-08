<script setup lang="ts">
import type { Activity } from "@/models/activity.model";
import { defineProps } from "vue";

defineProps<{
  model: Activity;
}>();

import { eventBus } from "@/main";

  // Emit the event to the event bus so that the parent component can handle it
  function handleDetailledActivityClick(link: string) {
  // extract id from link
  const id = link.split("/").pop();
  eventBus.emit('detailledActivityClick', id);
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
