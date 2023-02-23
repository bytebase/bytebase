<template>
  <button
    v-bind="$attrs"
    class="text-sm"
    :class="[!hideIcon && 'btn-icon']"
    @click.prevent.stop="
      () => {
        if (requireConfirm) {
          state.showModal = true;
        } else {
          $emit('confirm');
        }
      }
    "
  >
    <template v-if="!hideIcon">
      <heroicons-outline:trash v-if="style == 'DELETE'" class="w-4 h-4" />
      <heroicons-outline:archive v-if="style == 'ARCHIVE'" class="w-4 h-4" />
      <heroicons-outline:reply v-if="style == 'RESTORE'" class="w-4 h-4" />
      <heroicons-outline:minus-circle
        v-if="style == 'DISABLE'"
        class="w-4 h-4"
      />
      <heroicons-outline:pencil v-if="style == 'EDIT'" class="w-4 h-4" />
      <heroicons-outline:duplicate v-if="style == 'CLONE'" class="w-4 h-4" />
    </template>
    <span v-if="buttonText" :class="[!hideIcon && 'ml-1']">
      {{ buttonText }}
    </span>
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

<script lang="ts" setup>
import { reactive, withDefaults } from "vue";
import { BBButtonConfirmStyle } from "./types";

withDefaults(
  defineProps<{
    style?: BBButtonConfirmStyle;
    buttonText?: string;
    requireConfirm?: boolean;
    okText?: string;
    confirmTitle?: string;
    confirmDescription?: string;
    hideIcon?: boolean;
  }>(),
  {
    style: "DELETE",
    buttonText: "",
    requireConfirm: false,
    okText: "Delete",
    confirmTitle: "Are you sure to delete?",
    confirmDescription: "You cannot undo this action",
    hideIcon: false,
  }
);

defineEmits<{
  (event: "confirm"): void;
}>();

const state = reactive({
  showModal: false,
});
</script>
