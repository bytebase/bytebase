<template>
  <BBGrid
    :column-list="columns"
    :data-source="sortedSheetList"
    :show-placeholder="true"
    :ready="!isLoading"
    row-key="name"
    @click-row="handleSheetClick"
  >
    <template #item="{ item: sheet }: BBGridRow<Sheet>">
      <div class="bb-grid-cell">
        {{ sheet.title }}
      </div>
      <div class="bb-grid-cell">
        <ProjectV1Name :project="projectForSheet(sheet)" :link="false" />
      </div>
      <div class="bb-grid-cell">
        {{ visibilityDisplayName(sheet.visibility) }}
      </div>
      <div v-if="showCreator" class="bb-grid-cell">
        {{ creatorForSheet(sheet) }}
      </div>
      <div class="bb-grid-cell">
        <HumanizeDate :date="sheet.updateTime" />
      </div>
      <div class="bb-grid-cell" @click.stop>
        <Dropdown :sheet="sheet" :view="view" />
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, BBGridRow, BBGridColumn } from "@/bbkit";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { ProjectV1Name } from "@/components/v2";
import { useUserStore, useProjectV1Store } from "@/store";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { Sheet_Visibility } from "@/types/proto/v1/sheet_service";
import { extractProjectResourceName } from "@/utils";
import { SheetViewMode, useSheetContextByView, Dropdown } from "../../Sheet";

const props = defineProps<{
  view: SheetViewMode;
}>();

const emit = defineEmits<{
  (event: "select-sheet", sheet: Sheet): void;
}>();

const { t } = useI18n();
const projectStore = useProjectV1Store();
const userStore = useUserStore();
const { isInitialized, isLoading, sheetList, fetchSheetList } =
  useSheetContextByView(props.view);

const showCreator = computed(() => {
  return props.view === "shared" || props.view === "starred";
});

const handleSheetClick = (sheet: Sheet) => {
  emit("select-sheet", sheet);
};

const columns = computed(() => {
  const NAME: BBGridColumn = {
    title: t("common.name"),
    width: "2fr",
  };
  const PROJECT: BBGridColumn = {
    title: t("common.project"),
    width: "minmax(auto, 1fr)",
  };
  const VISIBILITY: BBGridColumn = {
    title: t("common.visibility"),
    width: "minmax(auto, 1fr)",
  };
  const CREATOR: BBGridColumn = {
    title: t("common.creator"),
    width: "minmax(auto, 1fr)",
  };
  const UPDATED: BBGridColumn = {
    title: t("common.updated-at"),
    width: "minmax(auto, 1fr)",
  };
  const OPERATION: BBGridColumn = {
    title: "",
    width: "auto",
  };
  const columns = [NAME, PROJECT, VISIBILITY];
  if (showCreator.value) {
    columns.push(CREATOR);
  }
  columns.push(UPDATED, OPERATION);
  return columns;
});

const sortedSheetList = computed(() => {
  return orderBy<Sheet>(sheetList.value, [(sheet) => sheet.title], ["asc"]);
});

const projectForSheet = (sheet: Sheet) => {
  const project = extractProjectResourceName(sheet.name);
  return projectStore.getProjectByName(`projects/${project}`);
};

const creatorForSheet = (sheet: Sheet) => {
  return userStore.getUserByIdentifier(sheet.creator)?.title ?? sheet.creator;
};

const visibilityDisplayName = (visibility: Sheet_Visibility) => {
  switch (visibility) {
    case Sheet_Visibility.VISIBILITY_PRIVATE:
      return t("sql-editor.private");
    case Sheet_Visibility.VISIBILITY_PROJECT:
      return t("common.project");
    case Sheet_Visibility.VISIBILITY_PUBLIC:
      return t("sql-editor.public");
    default:
      return "";
  }
};

onMounted(() => {
  if (!isInitialized.value) {
    fetchSheetList();
  }
});
</script>
