<template>
  <div
    class="flex flex-row items-center overflow-hidden gap-x-1 p-1 mx-1 rounded hover:bg-accent/10 cursor-pointer"
  >
    <FileCodeIcon class="w-4 h-4" />
    <div class="flex-1 flex flex-row-items-center truncate">
      <NPerformantEllipsis>
        {{ worksheet.title }}
      </NPerformantEllipsis>
    </div>
    <Dropdown
      :sheet="worksheet"
      :view="props.view"
      :secondary="true"
      :unsaved="unsaved"
    />
  </div>
</template>

<script setup lang="ts">
import { FileCodeIcon } from "lucide-vue-next";
import { NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { useSQLEditorTabStore } from "@/store";
import type { Worksheet } from "@/types/proto/v1/worksheet_service";
import type { SheetViewMode } from "@/views/sql-editor/Sheet";
import { Dropdown } from "@/views/sql-editor/Sheet";

const props = defineProps<{
  view: SheetViewMode;
  worksheet: Worksheet;
  keyword?: string;
}>();

const tabStore = useSQLEditorTabStore();
const unsaved = computed(() => {
  const tab = tabStore.tabList.find(
    (tab) => tab.sheet === props.worksheet.name
  );

  return tab && (tab.status === "DIRTY" || tab.status === "NEW");
});
</script>
