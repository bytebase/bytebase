<template>
  <button
    class="items-center flex justify-center btn-icon-primary"
    @click.prevent="$emit('add')"
  >
    <svg
      class="w-6 h-6"
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M12 6v6m0 0v6m0-6h6m-6 0H6"
      ></path>
    </svg>
  </button>
</template>

<script lang="ts">
export default {
  name: "BBButtonAdd",
  props: {},
  emits: ["add"],
};
</script>
