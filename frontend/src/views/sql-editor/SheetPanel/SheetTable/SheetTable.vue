<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="sortedSheetList"
    :loading="isLoading"
    :striped="true"
    :bordered="true"
    :row-key="(sheet: Worksheet) => sheet.name"
    :row-props="getRowProps"
  />
</template>

<script lang="tsx" setup>
import { escape, orderBy } from "lodash-es";
import { NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { ProjectV1Name } from "@/components/v2";
import { useUserStore, useProjectV1Store } from "@/store";
import { getDateForPbTimestamp } from "@/types";
import type { Worksheet } from "@/types/proto/v1/worksheet_service";
import { Worksheet_Visibility } from "@/types/proto/v1/worksheet_service";
import { getHighlightHTMLByRegExp } from "@/utils";
import type { SheetViewMode } from "../../Sheet";
import { useSheetContextByView, Dropdown } from "../../Sheet";
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

const columns = computed((): DataTableColumn<Worksheet>[] => {
  const cols: DataTableColumn<Worksheet>[] = [
    {
      title: t("common.name"),
      key: "name",
      render: (sheet) => <div v-html={titleHTML(sheet)} />,
    },
    {
      title: t("sql-editor.sheet.connection"),
      key: "connection",
      render: (sheet) => <SheetConnection sheet={sheet} />,
    },
    {
      title: t("common.project"),
      key: "project",
      render: (sheet) => (
        <ProjectV1Name project={projectForSheet(sheet)} link={false} />
      ),
    },
    {
      title: t("common.visibility"),
      key: "visibility",
      width: 150,
      render: (sheet) => visibilityDisplayName(sheet.visibility),
    },
  ];

  if (showCreator.value) {
    cols.push({
      title: t("common.creator"),
      key: "creator",
      render: (sheet) => creatorForSheet(sheet),
    });
  }

  cols.push(
    {
      title: t("common.updated-at"),
      key: "updatedAt",
      width: 180,
      render: (sheet) => (
        <HumanizeDate date={getDateForPbTimestamp(sheet.updateTime)} />
      ),
    },
    {
      title: "",
      key: "operations",
      width: 50,
      render: (sheet) => <Dropdown sheet={sheet} view={props.view} />,
    }
  );

  return cols;
});

const getRowProps = (sheet: Worksheet) => {
  return {
    style: "cursor: pointer;",
    onClick: () => {
      handleSheetClick(sheet);
    },
  };
};

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
    case Worksheet_Visibility.VISIBILITY_PROJECT_READ:
      return t("sql-editor.project-read");
    case Worksheet_Visibility.VISIBILITY_PROJECT_WRITE:
      return t("sql-editor.project-write");
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
