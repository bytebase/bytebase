<template>
  <BBGrid
    :column-list="columns"
    :data-source="sortedSheetList"
    :show-placeholder="true"
    :ready="!isLoading"
    row-key="name"
    @click-row="handleSheetClick"
  >
    <template #item="{ item: sheet }: BBGridRow<Worksheet>">
      <!-- eslint-disable-next-line vue/no-v-html -->
      <div class="bb-grid-cell" v-html="titleHTML(sheet)"></div>
      <div class="bb-grid-cell">
        <SheetConnection :sheet="sheet" />
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
import { escape, orderBy } from "lodash-es";
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, BBGridRow, BBGridColumn } from "@/bbkit";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { ProjectV1Name } from "@/components/v2";
import { useUserStore, useProjectV1Store } from "@/store";
import {
  Worksheet,
  Worksheet_Visibility,
} from "@/types/proto/v1/worksheet_service";
import { getHighlightHTMLByRegExp } from "@/utils";
import { SheetViewMode, useSheetContextByView, Dropdown } from "../../Sheet";
import SheetConnection from "./SheetConnection.vue";

const props = defineProps<{
  view: SheetViewMode;
  keyword?: string;
}>();

const emit = defineEmits<{
  (event: "select-sheet", sheet: Worksheet): void;
}>();

const { t } = useI18n();
const projectStore = useProjectV1Store();
const userStore = useUserStore();
const { isInitialized, isLoading, sheetList, fetchSheetList } =
  useSheetContextByView(props.view);

const showCreator = computed(() => {
  return props.view === "shared" || props.view === "starred";
});

const handleSheetClick = (sheet: Worksheet) => {
  emit("select-sheet", sheet);
};

const columns = computed(() => {
  const NAME: BBGridColumn = {
    title: t("common.name"),
    width: "2fr",
  };
  const CONNECTION: BBGridColumn = {
    title: t("sql-editor.sheet.connection"),
    width: "minmax(auto, 2fr)",
  };
  const PROJECT: BBGridColumn = {
    title: t("common.project"),
    width: "minmax(auto, 1fr)",
  };
  const VISIBILITY: BBGridColumn = {
    title: t("common.visibility"),
    width: "minmax(auto, 8rem)",
  };
  const CREATOR: BBGridColumn = {
    title: t("common.creator"),
    width: "minmax(auto, 1fr)",
  };
  const UPDATED: BBGridColumn = {
    title: t("common.updated-at"),
    width: "minmax(auto, 10rem)",
  };
  const OPERATION: BBGridColumn = {
    title: "",
    width: "auto",
  };
  const columns = [NAME, CONNECTION, PROJECT, VISIBILITY];
  if (showCreator.value) {
    columns.push(CREATOR);
  }
  columns.push(UPDATED, OPERATION);
  return columns;
});

const filteredList = computed(() => {
  const keyword = props.keyword?.toLowerCase()?.trim();
  if (!keyword) return sheetList.value;
  return sheetList.value.filter((sheet) => {
    return sheet.title.toLowerCase().includes(keyword);
  });
});

const sortedSheetList = computed(() => {
  return orderBy<Worksheet>(
    filteredList.value,
    [(sheet) => sheet.title],
    ["asc"]
  );
});

const projectForSheet = (sheet: Worksheet) => {
  return projectStore.getProjectByName(sheet.project);
};

const creatorForSheet = (sheet: Worksheet) => {
  return userStore.getUserByIdentifier(sheet.creator)?.title ?? sheet.creator;
};

const visibilityDisplayName = (visibility: Worksheet_Visibility) => {
  switch (visibility) {
    case Worksheet_Visibility.VISIBILITY_PRIVATE:
      return t("sql-editor.private");
    case Worksheet_Visibility.VISIBILITY_PROJECT:
      return t("common.project");
    case Worksheet_Visibility.VISIBILITY_PUBLIC:
      return t("sql-editor.public");
    default:
      return "";
  }
};

const titleHTML = (sheet: Worksheet) => {
  const kw = props.keyword?.toLowerCase().trim();
  const { title } = sheet;

  if (!kw) {
    return escape(title);
  }

  return getHighlightHTMLByRegExp(
    escape(title),
    escape(kw),
    false /* !caseSensitive */
  );
};

onMounted(() => {
  if (!isInitialized.value) {
    fetchSheetList();
  }
});
</script>
