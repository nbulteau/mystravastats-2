<template>
  <button
    class="btn btn-primary"
    @click="toggle"
  >
    {{ buttonText }} {{ isOn ? 'on' : 'off' }}
  </button>
</template>
  
  <script setup lang="ts">
  import { ref, watch, defineEmits } from 'vue';

  const emit = defineEmits(['update:modelValue']);
  defineProps<{ buttonText: string }>();
  const isOn = ref(false);
  
  const toggle = () => {
    isOn.value = !isOn.value;
  };

  watch(isOn, (newVal) => {
  emit('update:modelValue', newVal);
});
  </script>
  
  <style scoped>

  </style>