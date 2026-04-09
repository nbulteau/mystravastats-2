<template>
  <button
    class="btn btn-primary"
    @click="toggle"
  >
    {{ buttonText }} {{ modelValue ? 'on' : 'off' }}
  </button>
</template>
  
<script setup lang="ts">
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>();

const props = defineProps<{
  buttonText: string;
  modelValue: boolean;
}>();

const toggle = () => {
  emit('update:modelValue', !props.modelValue);
};
</script>
  
<style scoped>

</style>
