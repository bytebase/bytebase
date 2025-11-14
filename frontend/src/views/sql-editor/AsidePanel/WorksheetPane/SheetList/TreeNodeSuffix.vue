
<template>
  <template v-if="view === 'draft'">
    <XIcon
      v-if="node.worksheet"
      class="w-4 h-auto text-gray-600"
      @click="handleDeleteDraft"
    />
  </template>
  <MoreHorizontalIcon
    v-else-if="!node.worksheet"
    class="w-4 h-auto text-gray-600"
    @click="handleContextMenuShow"
  />
  <div
    v-else-if="worksheet"
    class="inline-flex gap-1"
  >
    <NTooltip
      v-if="
        worksheet.visibility === Worksheet_Visibility.PROJECT_READ ||
        worksheet.visibility === Worksheet_Visibility.PROJECT_WRITE
      "
    >
      <template #trigger>
        <UsersIcon
          class="w-4 text-gray-400"
          @click="handleSharePanelShow"
        />
      </template>
      <div>
        <div>
          {{ t("common.visibility") }}{{ ": " }}{{ visibilityDisplayName(worksheet.visibility) }}
        </div>
        <div
          v-if="!isWorksheetCreator(worksheet)"
        >
          {{ t("common.creator") }}{{ ": " }}{{ creatorForSheet(worksheet) }}
        </div>
      </div>
    </NTooltip>
    <StarIcon
      :class="`w-4 h-auto text-gray-400 ${worksheet.starred ? 'text-yellow-400' : ''}`"
      @click="handleToggleStar"
    />
    <MoreHorizontalIcon
      class="w-4 h-auto text-gray-600"
      @click="handleContextMenuShow"
    />
  </div>
</template>

<script setup lang="ts">
import {
  MoreHorizontalIcon,
  StarIcon,
  UsersIcon,
  XIcon,
} from "lucide-vue-next";
import { computed } from "vue";
import { NTooltip } from "naive-ui";
import { t } from "@/plugins/i18n";
import { useUserStore, useWorkSheetStore, useCurrentUserV1, useSQLEditorTabStore, useTabViewStateStore } from "@/store";
import {
  type Worksheet,
  Worksheet_Visibility,
} from "@/types/proto-es/v1/worksheet_service_pb";
import type { WorsheetFolderNode, SheetViewMode } from "@/views/sql-editor/Sheet";

const props = defineProps<{
  node: WorsheetFolderNode;
  view: SheetViewMode;
}>();

const emit = defineEmits<{
  (e: "contextMenuShow", event: MouseEvent, node: WorsheetFolderNode): void;
  (e: "sharePanelShow", event: MouseEvent, node: WorsheetFolderNode): void;
  (e: "toggleStar", worksheet: Worksheet): void;
}>();

const userStore = useUserStore();
const worksheetStore = useWorkSheetStore();
const me = useCurrentUserV1();
const { removeViewState } = useTabViewStateStore();
const tabStore = useSQLEditorTabStore();

const worksheet = computed(() => {
  if (!props.node.worksheet) {
    return undefined;
  }
  return worksheetStore.getWorksheetByName(props.node.worksheet.name);
});

const visibilityDisplayName = (visibility: Worksheet_Visibility) => {
  switch (visibility) {
    case Worksheet_Visibility.PRIVATE:
      return t("sql-editor.private");
    case Worksheet_Visibility.PROJECT_READ:
      return t("sql-editor.project-read");
    case Worksheet_Visibility.PROJECT_WRITE:
      return t("sql-editor.project-write");
    default:
      return "";
  }
};

const creatorForSheet = (sheet: Worksheet) => {
  return (
    userStore.getUserByIdentifier(sheet.creator)?.title ?? sheet.creator
  );
};

const isWorksheetCreator = (worksheet: Worksheet) => {
  return worksheet.creator === `users/${me.value.email}`;
};

const handleDeleteDraft = () => {
  if (props.node.worksheet && props.node.worksheet.name) {
    const draft = tabStore.tabList.find((t) => t.id === props.node.worksheet!.name);
    if (draft) {
      tabStore.removeTab(draft);
    }
    removeViewState(props.node.worksheet.name);
  }
};

const handleContextMenuShow = (e: MouseEvent) => {
  emit("contextMenuShow", e, props.node);
};

const handleSharePanelShow = (e: MouseEvent) => {
  emit("sharePanelShow", e, props.node);
};

const handleToggleStar = () => {
  if (worksheet.value) {
    emit("toggleStar", worksheet.value);
  }
};
</script>
