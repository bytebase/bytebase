<template>
  <NModal
    :show="show"
    preset="dialog"
    :type="type"
    :title="title"
    :content="description"
    :negative-text="negativeText"
    :positive-text="positiveText"
    @update:show="$emit('update:show', $event)"
    @positive-click="() => $emit('ok')"
    @negative-click="() => $emit('cancel')"
  >
    <slot name="default" />
  </NModal>
</template>

<script lang="ts" setup>
import { NModal } from "naive-ui";
import { withDefaults, computed } from "vue";
import { useI18n } from "vue-i18n";

const props = withDefaults(
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
    okText: "bbkit.common.ok",
    cancelText: "bbkit.common.cancel",
    payload: undefined,
  }
);

defineEmits<{
  (event: "ok"): void;
  (event: "cancel"): void;
  (event: "update:show", val: boolean): void;
}>();

const { t, te } = useI18n();

const negativeText = computed(() => {
  const { cancelText } = props;
  if (te(cancelText)) return t(cancelText);
  return cancelText;
});

const positiveText = computed(() => {
  const { okText } = props;
  if (te(okText)) return t(okText);
  return okText;
});
</script>
