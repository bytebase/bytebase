
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
    v-else-if="worksheetLite"
    class="inline-flex gap-1"
  >
    <NTooltip
      v-if="
        worksheetLite.visibility === Worksheet_Visibility.PROJECT_READ ||
        worksheetLite.visibility === Worksheet_Visibility.PROJECT_WRITE
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
          {{ t("common.visibility") }}{{ ": " }}{{ visibilityDisplayName(worksheetLite.visibility) }}
        </div>
        <div
          v-if="!isWorksheetCreator(worksheetLite.creator)"
        >
          {{ t("common.creator") }}{{ ": " }}{{ creatorForSheet(worksheetLite.creator) }}
        </div>
      </div>
    </NTooltip>
    <StarIcon
      :class="`w-4 h-auto text-gray-400 ${worksheetLite.starred ? 'text-yellow-400' : ''}`"
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
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { t } from "@/plugins/i18n";
import {
  useCurrentUserV1,
  useSQLEditorTabStore,
  useTabViewStateStore,
  useUserStore,
  useWorkSheetStore,
} from "@/store";
import { Worksheet_Visibility } from "@/types/proto-es/v1/worksheet_service_pb";
import type {
  SheetViewMode,
  WorksheetFolderNode,
} from "@/views/sql-editor/Sheet";

const props = defineProps<{
  node: WorksheetFolderNode;
  view: SheetViewMode;
}>();

const emit = defineEmits<{
  (e: "contextMenuShow", event: MouseEvent, node: WorksheetFolderNode): void;
  (e: "sharePanelShow", event: MouseEvent, node: WorksheetFolderNode): void;
  (e: "toggleStar", item: { worksheet: string; starred: boolean }): void;
}>();

const userStore = useUserStore();
const worksheetStore = useWorkSheetStore();
const me = useCurrentUserV1();
const { removeViewState } = useTabViewStateStore();
const tabStore = useSQLEditorTabStore();

const worksheetLite = computed(() => {
  if (!props.node.worksheet) {
    return undefined;
  }
  const sheet = worksheetStore.getWorksheetByName(props.node.worksheet.name);
  if (!sheet) {
    return undefined;
  }
  return {
    name: sheet.name,
    starred: sheet.starred,
    visibility: sheet.visibility,
    creator: sheet.creator,
  };
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

const creatorForSheet = (creator: string) => {
  return userStore.getUserByIdentifier(creator)?.title ?? creator;
};

const isWorksheetCreator = (creator: string) => {
  return creator === `users/${me.value.email}`;
};

const handleDeleteDraft = () => {
  if (props.node.worksheet && props.node.worksheet.name) {
    const draft = tabStore.tabList.find(
      (t) => t.id === props.node.worksheet!.name
    );
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
  if (worksheetLite.value) {
    emit("toggleStar", {
      worksheet: worksheetLite.value.name,
      starred: !worksheetLite.value.starred,
    });
  }
};
</script>
