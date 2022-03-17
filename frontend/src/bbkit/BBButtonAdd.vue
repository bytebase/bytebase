<template>
  <button
    class="items-center flex justify-center btn-icon-primary"
    @click.prevent="$emit('add')"
  >
    <heroicons-solid:plus class="w-6 h-6" />
  </button>
</template>

<script lang="ts" setup>
defineEmits<{
  (event: "add"): void;
}>();
</script>
