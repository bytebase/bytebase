<template>
  <NButton
    v-if="isSupported && !isEmpty"
    v-bind="$attrs"
    :text="text"
    :size="size"
    :disabled="disabled"
    @click="handleCopy"
  >
    <template #icon>
      <CopyIcon />
    </template>
    <slot />
  </NButton>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import { CopyIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

const props = withDefaults(
  defineProps<{
    content: string | (() => string);
    text?: boolean;
    disabled?: boolean;
    size?: "tiny" | "small" | "medium" | "large";
  }>(),
  {
    text: true,
    disabled: false,
    size: "tiny",
  }
);

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});
const { t } = useI18n();

const isEmpty = computed(() => {
  if (typeof props.content === "function") {
    return false;
  }
  return !props.content;
});

const handleCopy = () => {
  if (!isSupported.value) {
    return;
  }
  let value = "";
  if (typeof props.content === "function") {
    value = props.content();
  } else {
    value = props.content;
  }
  copyTextToClipboard(value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  });
};
</script>
