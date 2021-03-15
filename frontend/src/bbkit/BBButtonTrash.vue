<template>
  <button
    class="btn-icon text-sm"
    @click.prevent="
      () => {
        if (requireConfirm) {
          state.showDeleteModal = true;
        } else {
          $emit('confirm');
        }
      }
    "
  >
    <svg
      class="w-4 h-4"
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
      ></path>
    </svg>
    <span class="ml-1">{{ buttonText }}</span>
  </button>
  <BBAlert
    v-if="state.showDeleteModal"
    :style="'CRITICAL'"
    :okText="okText"
    :title="confirmTitle"
    :description="confirmDescription"
    @ok="
      () => {
        state.showDeleteModal = false;
        $emit('confirm');
      }
    "
    @cancel="state.showDeleteModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { reactive } from "vue";

export default {
  name: "BBButtonTrash",
  emits: ["confirm"],
  props: {
    buttonText: {
      default: "",
      type: String,
    },
    requireConfirm: {
      default: false,
      type: Boolean,
    },
    okText: {
      default: "Delete",
      type: String,
    },
    confirmTitle: {
      default: "Are you sure to delete?",
      type: String,
    },
    confirmDescription: {
      default: "You cannot undo this action",
      type: String,
    },
  },
  setup(props, ctx) {
    const state = reactive({
      showDeleteModal: false,
    });

    return {
      state,
    };
  },
};
</script>
