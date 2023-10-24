<template>
  <NDataTable
    v-bind="$attrs"
    :loading="isLoading"
    :bordered="false"
    :columns="dataTableColumns"
    :data="dataTableRows"
    :row-key="rowKey"
    :row-props="rowProps"
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NDataTable } from "naive-ui";
import { computed, ref, watch, h } from "vue";
import { useI18n } from "vue-i18n";
import BranchBaseline from "@/components/Branch/BranchBaseline.vue";
import {
  useChangeHistoryStore,
  useDatabaseV1Store,
  useProjectV1Store,
  useUserStore,
} from "@/store";
import { getProjectAndSchemaDesignSheetId } from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "@/types";
import {
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";

type BranchRowData = {
  branch: SchemaDesign;
  name: string;
  branchName: string;
  projectName: string;
  baselineVersion: string;
  updatedTimeStr: string;
  children?: BranchRowData[];
};

const props = defineProps<{
  branches: SchemaDesign[];
  hideProjectColumn?: boolean;
  ready?: boolean;
}>();

const emit = defineEmits<{
  (event: "click", schemaDesign: SchemaDesign): void;
}>();

const { t } = useI18n();
const userV1Store = useUserStore();
const projectV1Store = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const changeHistoryStore = useChangeHistoryStore();
const isFetching = ref(true);

watch(
  () => props.branches,
  async () => {
    for (const branch of props.branches) {
      const database = await databaseStore.getOrFetchDatabaseByName(
        branch.baselineDatabase
      );
      if (
        database &&
        branch.baselineChangeHistoryId &&
        branch.baselineChangeHistoryId !== String(UNKNOWN_ID)
      ) {
        const changeHistoryName = `${database.name}/changeHistories/${branch.baselineChangeHistoryId}`;
        await changeHistoryStore.getOrFetchChangeHistoryByName(
          changeHistoryName
        );
      }
    }
    isFetching.value = false;
  },
  {
    deep: true,
    immediate: true,
  }
);

const isLoading = computed(() => {
  return isFetching.value || !props.ready;
});

const dataTableRows = computed(() => {
  const parentBranches = props.branches.filter((branch) => {
    return branch.type === SchemaDesign_Type.MAIN_BRANCH;
  });
  const parentRows: BranchRowData[] = parentBranches.map((branch) => {
    const [projectName] = getProjectAndSchemaDesignSheetId(branch.name);
    const project = projectV1Store.getProjectByName(`projects/${projectName}`);
    const database = databaseStore.getDatabaseByName(branch.baselineDatabase);
    const changeHistory =
      branch.baselineChangeHistoryId &&
      branch.baselineChangeHistoryId !== String(UNKNOWN_ID)
        ? changeHistoryStore.getChangeHistoryByName(
            `${database.name}/changeHistories/${branch.baselineChangeHistoryId}`
          )
        : undefined;
    const baselineVersion = `(${database.effectiveEnvironmentEntity.title}) ${
      database.databaseName
    } @${changeHistory ? changeHistory.version : "Previously latest schema"}`;

    return {
      branch: branch,
      name: branch.title,
      branchName: branch.title,
      projectName: project.title,
      baselineVersion: baselineVersion,
      updatedTimeStr: getUpdatedTimeStr(branch),
      children: [],
    };
  });
  const childBranches = props.branches.filter((branch) => {
    return branch.type === SchemaDesign_Type.PERSONAL_DRAFT;
  });
  for (const childBranch of childBranches) {
    const parentRow = parentRows.find(
      (row) => row.branch.name === childBranch.baselineSheetName
    );
    if (parentRow) {
      parentRow.children?.push({
        branch: childBranch,
        name: childBranch.title,
        branchName: `${parentRow.branchName}/${childBranch.title}`,
        // Child branch does not show project name.
        projectName: "",
        baselineVersion: "",
        updatedTimeStr: getUpdatedTimeStr(childBranch),
      });
    }
  }

  return parentRows;
});

const dataTableColumns = computed(() => {
  const BRANCH_NAME_COLUMN = {
    title: t("common.branch"),
    key: "branchName",
  };
  const PROJECT_NAME_COLUMN = {
    title: t("common.project"),
    key: "projectName",
  };
  const BASELINE_VERSION_COLUMN = {
    title: t("schema-designer.baseline-version"),
    key: "baselineVersion",
    render: (row: BranchRowData) => {
      return h(BranchBaseline, {
        branch: row.branch,
        showInstanceIcon: true,
      });
    },
  };
  const UPDATED_TIME_COLUMN = {
    title: t("common.updated"),
    key: "updatedTimeStr",
  };

  return [
    BRANCH_NAME_COLUMN,
    !props.hideProjectColumn ? PROJECT_NAME_COLUMN : undefined,
    BASELINE_VERSION_COLUMN,
    UPDATED_TIME_COLUMN,
  ].filter((column) => column);
});

const rowKey = (row: BranchRowData) => {
  return row.branch.name;
};

const rowProps = (row: BranchRowData) => {
  return {
    class: "cursor-pointer hover:bg-gray-100",
    onClick: (event: MouseEvent) => {
      const targetElement = event.target as HTMLElement;
      const triggerElement = targetElement.closest(
        "div.n-data-table-expand-trigger"
      );
      // Only emit click event when user clicks on the row but not the expand trigger.
      if (!triggerElement) {
        emit("click", row.branch);
      }
    },
  };
};

const getUpdatedTimeStr = (branch: SchemaDesign) => {
  const updater = userV1Store.getUserByEmail(branch.updater.split("/")[1]);
  const updatedTimeStr = t("schema-designer.message.updated-time-by-user", {
    time: dayjs
      .duration((branch.updateTime ?? new Date()).getTime() - Date.now())
      .humanize(true),
    user: updater?.title,
  });
  return updatedTimeStr;
};
</script>

<style lang="postcss">
.n-data-table-expand-trigger {
  @apply !w-5 !h-5 inline-flex justify-center items-center translate-y-0.5 rounded hover:bg-white hover:shadow;
}
.n-data-table-expand-trigger > .n-base-icon {
  @apply !w-5 !h-5 flex flex-row justify-center items-center;
}
.n-data-table .n-data-table-td {
  background-color: transparent !important;
}
</style>
