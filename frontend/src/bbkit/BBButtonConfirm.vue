<template>
  <NButton
    v-bind="$attrs"
    class="text-sm"
    :type="type"
    :class="[!state.hideIcon && 'btn-icon']"
    @click.prevent.stop="
      () => {
        if (state.requireConfirm) {
          state.showModal = true;
        } else {
          $emit('confirm');
        }
      }
    "
  >
    <template v-if="!state.hideIcon">
      <heroicons-outline:trash v-if="state.style == 'DELETE'" class="w-4 h-4" />
      <heroicons-outline:archive
        v-if="state.style == 'ARCHIVE'"
        class="w-4 h-4"
      />
      <heroicons-outline:reply
        v-if="state.style == 'RESTORE'"
        class="w-4 h-4"
      />
      <heroicons-outline:minus-circle
        v-if="state.style == 'DISABLE'"
        class="w-4 h-4"
      />
      <heroicons-outline:pencil v-if="state.style == 'EDIT'" class="w-4 h-4" />
      <heroicons-outline:duplicate
        v-if="state.style == 'CLONE'"
        class="w-4 h-4"
      />
    </template>
    <span v-if="state.buttonText" :class="[!state.hideIcon && 'ml-1']">
      {{ state.buttonText }}
    </span>
  </NButton>
  <BBAlert
    v-if="state.showModal"
    :type="
      state.style == 'DELETE' ||
      state.style == 'ARCHIVE' ||
      state.style == 'DISABLE'
        ? 'warning'
        : 'info'
    "
    :ok-text="state.okText"
    :title="state.confirmTitle"
    :description="state.confirmDescription"
    @ok="
      () => {
        state.showModal = false;
        $emit('confirm');
      }
    "
    @cancel="state.showModal = false"
  >
    <slot name="default" />
  </BBAlert>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBButtonConfirmStyle } from "./types";

const props = withDefaults(
  defineProps<{
    type?: "text" | "default";
    style?: BBButtonConfirmStyle;
    buttonText?: string;
    requireConfirm?: boolean;
    okText?: string;
    confirmTitle?: string;
    confirmDescription?: string;
    hideIcon?: boolean;
  }>(),
  {
    type: "text",
    style: "DELETE",
    buttonText: "",
    requireConfirm: false,
    okText: "",
    confirmTitle: "",
    confirmDescription: "",
    hideIcon: false,
  }
);

defineEmits<{
  (event: "confirm"): void;
}>();

const { t } = useI18n();

const state = reactive({
  // computed props with default i18n values.
  style: props.style,
  buttonText: props.buttonText,
  requireConfirm: props.requireConfirm,
  okText: props.okText || t("common.delete"),
  confirmTitle: props.confirmTitle || t("bbkit.confirm-button.sure-to-delete"),
  confirmDescription:
    props.confirmDescription || t("bbkit.confirm-button.cannot-undo"),
  hideIcon: props.hideIcon,
  // local state.
  showModal: false,
});
</script>
