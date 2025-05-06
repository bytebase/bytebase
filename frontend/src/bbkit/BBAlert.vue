<template>
  <NModal
    :show="show"
    preset="dialog"
    :type="type"
    :title="title"
    :content="description"
    :negative-text="cancelText"
    :positive-text="okText"
    @update:show="$emit('update:show', $event)"
    @positive-click="() => $emit('ok')"
    @negative-click="() => $emit('cancel')"
  >
    <slot name="default" />
  </NModal>
</template>

<script lang="ts" setup>
import { NModal } from "naive-ui";
import { t } from "@/plugins/i18n";

withDefaults(
  defineProps<{
    show: boolean;
    type: "info" | "warning";
    title: string;
    description?: string;
    okText?: string;
    cancelText?: string;
  }>(),
  {
    type: "info",
    description: "",
    okText: () => t("common.ok"),
    cancelText: () => t("common.cancel"),
    payload: undefined,
  }
);

defineEmits<{
  (event: "ok"): void;
  (event: "cancel"): void;
  (event: "update:show", val: boolean): void;
}>();
</script>
