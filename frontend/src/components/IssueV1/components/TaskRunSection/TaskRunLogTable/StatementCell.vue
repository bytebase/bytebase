<template>
  <div class="text-sm group">
    <TextOverflowPopover
      v-if="statement"
      :content="statement.trim()"
      :max-length="100"
      :max-popover-content-length="1000"
      :line-wrap="false"
      :line-break-replacer="' '"
      code-class="relative"
      placement="top"
    >
      <template #default="{ displayContent }">
        <span class="flex-1 flex flex-row justify-start items-center gap-x-1">
          <span class="line-clamp-1">
            {{ displayContent }}
          </span>
          <NButton
            text
            size="tiny"
            class="invisible group-hover:visible"
            @click="copyStatement"
          >
            <template #icon>
              <CopyIcon class="w-3 h-3" />
            </template>
          </NButton>
        </span>
      </template>
      <template #popover-header>
        <div class="absolute bottom-1 right-1">
          <NButton text size="tiny" @click="copyStatement">
            <template #icon>
              <CopyIcon class="w-3 h-3" />
            </template>
          </NButton>
        </div>
      </template>
    </TextOverflowPopover>
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import { CopyIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import { pushNotification } from "@/store";
import { TaskRunLogEntry_Type } from "@/types/proto/api/v1alpha/rollout_service";
import type { Sheet } from "@/types/proto/api/v1alpha/sheet_service";
import { extractSheetCommandByIndex, toClipboard } from "@/utils";
import type { FlattenLogEntry } from "./common";

const props = defineProps<{
  entry: FlattenLogEntry;
  sheet?: Sheet;
}>();

const { t } = useI18n();

const statement = computed(() => {
  const { entry, sheet } = props;
  if (entry.type === TaskRunLogEntry_Type.SCHEMA_DUMP && entry.schemaDump) {
    return "";
  }
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    if (!sheet) {
      return "";
    }
    if (!sheet.payload?.commands) {
      return "";
    }

    const { commandExecute } = entry;
    return extractSheetCommandByIndex(sheet, commandExecute.commandIndex) ?? "";
  }
  return "";
});

const copyStatement = () => {
  toClipboard(statement.value.trim()).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  });
};
</script>
