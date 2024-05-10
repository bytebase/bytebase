<template>
  <div class="text-sm group">
    <TextOverflowPopover
      v-if="statement"
      :content="statement.trim()"
      :max-length="100"
      :max-popover-content-length="1000"
      :line-wrap="false"
      :line-break-replacer="' '"
      content-class=""
      placement="top"
    >
      <template #default="{ displayContent }">
        <span class="flex-1 flex flex-row justify-start items-center gap-x-1">
          <span class="line-clamp-1">
            {{ displayContent }}
          </span>
          <NButton text size="tiny" class="invisible group-hover:visible">
            <template #icon>
              <CopyIcon class="w-3 h-3" />
            </template>
          </NButton>
        </span>
      </template>
    </TextOverflowPopover>
    <div v-else class="text-control-placeholder">-</div>
  </div>
</template>

<script setup lang="ts">
import { CopyIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import { TaskRunLogEntry_Type } from "@/types/proto/v1/rollout_service";
import type { Sheet } from "@/types/proto/v1/sheet_service";
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
    if (!sheet) {
      return "";
    }
    if (!sheet.payload?.commands) {
      return "";
    }

    const { commandExecute } = entry;
    return extractSheetCommandByIndex(sheet, commandExecute.commandIndex);
  }
  return "";
});
</script>
