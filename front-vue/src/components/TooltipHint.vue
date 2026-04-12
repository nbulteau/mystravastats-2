<script setup lang="ts">
const props = defineProps<{
  text: string;
}>();
</script>

<template>
  <span
    class="tooltip-hint"
    :title="props.text"
    tabindex="0"
    role="note"
    :aria-label="props.text"
  >
    ?
  </span>
</template>

<style scoped>
.tooltip-hint {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1rem;
  height: 1rem;
  margin-left: 0.35rem;
  border: 1px solid #ffc9b2;
  border-radius: 999px;
  color: #d44700;
  background: #fff6f1;
  font-size: 0.72rem;
  font-weight: 700;
  line-height: 1;
  cursor: help;
  user-select: none;
}

.tooltip-hint:focus-visible {
  outline: 2px solid rgba(252, 76, 2, 0.45);
  outline-offset: 1px;
}
</style>
