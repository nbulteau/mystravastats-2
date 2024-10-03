<template>
  <div class="date-cell">
    {{ formattedDate }}
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  value: string;
}>();

const options: Intl.DateTimeFormatOptions = {
  weekday: 'short',
  day: '2-digit',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit'
};

const formattedDate = computed(() => {
  const date = new Date(props.value);

  return new Intl.DateTimeFormat(navigator.language, options).format(date);
});

</script>

<style scoped>
.date-cell {
  text-align: right;
}
</style>