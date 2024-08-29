<template>
  <NButton
    v-bind="$attrs"
    class="text-sm"
    :text="type === 'text'"
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
  </NButton>
  <BBAlert
    v-model:show="state.showModal"
    :type="
      style == 'DELETE' || style == 'ARCHIVE' || style == 'DISABLE'
        ? 'warning'
        : 'info'
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
    <slot name="default" />
  </BBAlert>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import BBAlert from "./BBAlert.vue";
import type { BBButtonConfirmStyle } from "./types";

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
