<template>
  <div class="flex items-start w-full">
    <div class="flex-1 overflow-x-hidden">
      <MonacoEditor
        :content="code"
        :readonly="true"
        :auto-focus="false"
        :auto-height="{
          min: 48,
          max: 120,
          padding: 2,
        }"
        :options="{
          automaticLayout: true,
          fontSize: 12,
          lineHeight: 14,
          lineNumbers: 'off',
          wordWrap: 'true',
          scrollbar: {
            vertical: 'hidden',
            horizontal: 'hidden',
            useShadows: false,
            verticalScrollbarSize: 0,
            horizontalScrollbarSize: 0,
            alwaysConsumeMouseWheel: false,
          },
        }"
        class="border h-auto bg-white"
      />
    </div>
    <div class="flex flex-col gap-y-2 ml-0.5 mt-1">
      <NTooltip placement="right">
        <template #trigger>
          <button
            class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
            @click="handleExecute"
          >
            <heroicons:play-circle class="w-4 h-4" />
          </button>
        </template>
        <div class="whitespace-nowrap">
          {{ $t("common.run") }}
        </div>
      </NTooltip>
      <NTooltip placement="right">
        <template #trigger>
          <button
            class="inline-flex items-center justify-center hover:text-accent cursor-pointer"
            @click="handleCopy"
          >
            <heroicons:clipboard class="w-4 h-4" />
          </button>
        </template>
        <div class="whitespace-nowrap">
          {{ $t("common.copy") }}
        </div>
      </NTooltip>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { useI18n } from "vue-i18n";
import { MonacoEditor } from "@/components/MonacoEditor";
import { useAIContext } from "@/plugins/ai/logic";
import { pushNotification } from "@/store";
import { toClipboard } from "@/utils";

const props = defineProps<{
  code: string;
}>();

const { t } = useI18n();
const { events, showHistoryDialog } = useAIContext();

const handleExecute = () => {
  events.emit("apply-statement", {
    statement: props.code,
    run: true,
  });
  showHistoryDialog.value = false;
};

const handleCopy = () => {
  toClipboard(props.code).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("plugin.ai.statement-copied"),
    });
  });
};
</script>
