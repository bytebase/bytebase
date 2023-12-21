<template>
  <NModal
    :show="true"
    preset="dialog"
    :type="type"
    :title="title"
    :content="description"
    :negative-text="negativeText"
    :positive-text="positiveText"
    @positive-click="() => $emit('ok')"
    @negative-click="() => $emit('cancel')"
  />
</template>

<script lang="ts" setup>
import { NModal } from "naive-ui";
import { withDefaults, computed } from "vue";
import { useI18n } from "vue-i18n";

const props = withDefaults(
  defineProps<{
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

const emit = defineEmits<{
  (event: "ok"): void;
  (event: "cancel"): void;
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
