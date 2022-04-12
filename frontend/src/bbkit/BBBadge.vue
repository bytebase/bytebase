<template>
  <span
    :class="[
      'inline-flex rounded-full items-center py-0.5 px-2.5 text-sm font-medium',
      `bg-${theme}-100 text-${theme}-800`,
    ]"
  >
    {{ text }}
    <button
      v-if="canRemove"
      type="button"
      :class="[
        'flex-shrink-0 ml-0.5 h-4 w-4 rounded-full inline-flex items-center justify-center focus:outline-none',
        `text-${theme}-400 hover:bg-${theme}-200 hover:text-${theme}-500 focus:bg-${theme}-500`,
      ]"
      @click="$emit('remove')"
    >
      <svg class="h-2 w-2" stroke="currentColor" fill="none" viewBox="0 0 8 8">
        <path stroke-linecap="round" stroke-width="1.5" d="M1 1l6 6m0-6L1 7" />
      </svg>
    </button>
  </span>
</template>

<script lang="ts" setup>
const props = defineProps({
  text: {
    required: true,
    type: String,
  },
  canRemove: {
    required: false,
    type: Boolean,
    default: true,
  },
  theme: {
    required: false,
    type: String,
    default: "indigo",
  },
});

const emit = defineEmits(["remove"]);
</script>
