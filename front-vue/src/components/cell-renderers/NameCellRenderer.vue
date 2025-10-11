<script setup lang="ts">
import type {Activity} from "@/models/activity.model";
import {defineProps} from "vue";
import {eventBus} from "@/main";

defineProps<{
  model: Activity;
}>();


// Emit the event to the event bus so that the parent component can handle it
function handleDetailedActivityClick(id: string) {
  eventBus.emit("detailledActivityClick", id);
}
</script>

<template>
  <div>
    <a
        v-if="model.link"
        :href="model.link"
        target="_blank"
        class="btn btn-light"
    >
      <img
          src="@/assets/buttons/eye.png"
          alt="Info"
          width="16"
          height="16"
      >
    </a>

    <a
        href="#"
        class="activity-link"
        @click.prevent="handleDetailedActivityClick($props.model.id.toString())"
    >
      {{ model.name }}
    </a>
  </div>
</template>

<style scoped>
.activity-link {
  color: blue;
  text-decoration: underline;
}
</style>
