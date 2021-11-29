<template>
  <button
    class="btn-icon text-sm"
    @click.prevent="
      () => {
        if (requireConfirm) {
          state.showModal = true;
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
        v-if="style == 'DELETE'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
      ></path>
      <path
        v-if="style == 'ARCHIVE'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
      ></path>
      <path
        v-if="style == 'RESTORE'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"
      ></path>

      <path
        v-if="style == 'DISABLE'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M15 12H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z"
      ></path>
      <path
        v-if="style == 'EDIT'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
      ></path>
      <path
        v-if="style == 'CLONE'"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
      ></path>
    </svg>
    <span v-if="buttonText" class="ml-1">{{ buttonText }}</span>
  </button>
  <BBAlert
    v-if="state.showModal"
    :style="
      style == 'DELETE' || style == 'ARCHIVE' || style == 'DISABLE'
        ? 'CRITICAL'
        : 'INFO'
    "
    :ok-text="okText"
    :title="confirmTitle"
    :description="confirmDescription"
    @ok="
      () => {
        state.showModal = false;
        $emit('confirm');
      }
    "
    @cancel="state.showModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { PropType, reactive } from "vue";
import { BBButtonConfirmStyle } from "./types";

export default {
  name: "BBButtonConfirm",
  props: {
    style: {
      default: "DELETE",
      type: String as PropType<BBButtonConfirmStyle>,
    },
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
  emits: ["confirm"],
  setup() {
    const state = reactive({
      showModal: false,
    });

    return {
      state,
    };
  },
};
</script>
