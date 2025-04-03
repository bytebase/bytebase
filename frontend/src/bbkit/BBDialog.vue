<template>
  <BBModal
    :show="state.visible"
    :title="title"
    :subtitle="subtitle"
    :show-close="closable"
    :close-on-esc="closable"
    @close="onNegativeClick"
  >
    <slot name="default"></slot>

    <div class="pt-4 border-t border-block-border flex justify-end space-x-3">
      <NButton
        v-if="showNegativeBtn"
        size="small"
        @click.prevent="onNegativeClick"
      >
        {{ negativeText || $t("common.cancel") }}
      </NButton>
      <NButton
        v-if="showPositiveBtn"
        :type="type"
        size="small"
        data-label="bb-modal-confirm-button"
        @click.prevent="onPositiveClick"
      >
        {{ positiveText || $t("common.confirm") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { reactive } from "vue";
import type { Defer } from "@/utils";
import { defer } from "@/utils";
import BBModal from "./BBModal.vue";

withDefaults(
  defineProps<{
    title: string;
    subtitle?: string;
    closable?: boolean;
    showNegativeBtn?: boolean;
    showPositiveBtn?: boolean;
    negativeText?: string;
    positiveText?: string;
    type: "info" | "warning";
  }>(),
  {
    showNegativeBtn: true,
    showPositiveBtn: true,
    type: "info",
  }
);

const emit = defineEmits<{
  (event: "on-positive-click"): void;
  (event: "on-negative-click"): void;
}>();

const state = reactive({
  visible: false,
  defer: undefined as Defer<boolean> | undefined,
});

const open = () => {
  if (state.defer) {
    state.defer.reject(new Error("duplicated call"));
  }

  state.defer = defer<boolean>();
  state.visible = true;

  return state.defer.promise;
};

const onPositiveClick = () => {
  state.visible = false;
  state.defer?.resolve(true);

  emit("on-positive-click");
};

const onNegativeClick = () => {
  state.visible = false;
  state.defer?.resolve(false);

  emit("on-negative-click");
};

defineExpose({ open });
</script>
