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
          <CopyButton :content="statement.trim()" />
        </span>
      </template>
      <template #popover-header>
        <div class="absolute bottom-1 right-1">
          <CopyButton :content="statement.trim()" />
        </div>
      </template>
    </TextOverflowPopover>
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import { CopyButton } from "@/components/v2";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { extractSheetCommandByIndex } from "@/utils";
import type { FlattenLogEntry } from "./common";

const props = defineProps<{
  entry: FlattenLogEntry;
  sheet?: Sheet;
}>();

const statement = computed(() => {
  const { entry, sheet } = props;
  if (entry.type === TaskRunLogEntry_Type.SCHEMA_DUMP && entry.schemaDump) {
    return "";
  }
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    if (entry.commandExecute.kind === "statement") {
      return entry.commandExecute.statement;
    }
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
</script>
