<template>
  <NButton
    v-bind="$attrs"
    class="text-sm"
    :text="text"
    :size="size"
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
      <Trash2Icon v-if="type == 'DELETE'" class="w-4 h-4" />
      <ArchiveIcon v-if="type == 'ARCHIVE'" class="w-4 h-4" />
      <Undo2Icon v-if="type == 'RESTORE'" class="w-4 h-4" />
    </template>
    <span v-if="buttonText" :class="[!hideIcon && 'ml-1']">
      {{ buttonText }}
    </span>
  </NButton>
  <BBAlert
    v-model:show="state.showModal"
    :type="type == 'DELETE' || type == 'ARCHIVE' ? 'warning' : 'info'"
    :ok-text="okText"
    :title="confirmTitle"
    :description="confirmDescription"
    :positive-button-props="positiveButtonProps"
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
import { ArchiveIcon, Trash2Icon, Undo2Icon } from "lucide-vue-next";
import { NButton, type ButtonProps } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import BBAlert from "./BBAlert.vue";
import type { BBButtonConfirmType } from "./types";

const props = withDefaults(
  defineProps<{
    type?: BBButtonConfirmType;
    text?: boolean;
    buttonText?: string;
    positiveButtonProps?: ButtonProps | undefined;
    requireConfirm?: boolean;
    okText?: string;
    confirmTitle?: string;
    confirmDescription?: string;
    hideIcon?: boolean;
    size?: "tiny" | "small" | "medium" | "large";
  }>(),
  {
    type: "DELETE",
    text: true, // Default to display as a text button.
    buttonText: "",
    requireConfirm: false,
    okText: "",
    confirmTitle: "",
    confirmDescription: "",
    hideIcon: false,
    size: "medium",
  }
);

defineEmits<{
  (event: "confirm"): void;
}>();

const { t } = useI18n();

const state = reactive({
  showModal: false,
});

const okText = computed(() => {
  return props.okText || t("common.delete");
});
const confirmTitle = computed(() => {
  return props.confirmTitle || t("bbkit.confirm-button.sure-to-delete");
});
const confirmDescription = computed(() => {
  return props.confirmDescription || t("bbkit.confirm-button.cannot-undo");
});

defineExpose({
  showAlert: () => (state.showModal = true),
});
</script>
