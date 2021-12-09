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
    <heroicons-outline:trash v-if="style == 'DELETE'" class="w-4 h-4" />
    <heroicons-outline:archive v-if="style == 'ARCHIVE'" class="w-4 h-4" />
    <heroicons-outline:reply v-if="style == 'RESTORE'" class="w-4 h-4" />
    <heroicons-outline:minus-circle v-if="style == 'DISABLE'" class="w-4 h-4" />
    <heroicons-outline:pencil v-if="style == 'EDIT'" class="w-4 h-4" />
    <heroicons-outline:duplicate v-if="style == 'CLONE'" class="w-4 h-4" />
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
