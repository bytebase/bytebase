<template>
  <NButton
    v-if="isSupported && content"
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
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

const props = withDefaults(
  defineProps<{
    content: string;
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

const handleCopy = () => {
  if (!isSupported.value) {
    return;
  }
  copyTextToClipboard(props.content).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  });
};
</script>
