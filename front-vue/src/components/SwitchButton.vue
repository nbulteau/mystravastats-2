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
  .switch-button {
    width: 50px;
    height: 25px;
    background-color: #ccc;
    border-radius: 25px;
    position: relative;
    cursor: pointer;
  }
  
  .switch-button::before {
    content: '';
    position: absolute;
    width: 23px;
    height: 23px;
    background-color: white;
    border-radius: 50%;
    top: 1px;
    left: 1px;
    transition: transform 0.3s;
  }
  </style>