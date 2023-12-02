<template>
  <NDataTable
    v-bind="$attrs"
    :bordered="false"
    :columns="dataTableColumns"
    :data="dataTableRows"
    :row-key="rowKey"
    :row-props="rowProps"
    class="bb-branch-data-table"
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import BranchBaseline from "@/components/Branch/BranchBaseline.vue";
import { useDatabaseV1Store, useProjectV1Store, useUserStore } from "@/store";
import {
  getProjectAndSchemaDesignSheetId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { Branch } from "@/types/proto/v1/branch_service";

type BranchRowData = {
  branch: Branch;
  name: string;
  branchName: string;
  projectName: string;
  baselineVersion: string;
  updatedTimeStr: string;
  children?: BranchRowData[];
};

const props = defineProps<{
  branches: Branch[];
  hideProjectColumn?: boolean;
  ready?: boolean;
}>();

const emit = defineEmits<{
  (event: "click", branch: Branch): void;
}>();

const { t } = useI18n();
const userV1Store = useUserStore();
const projectV1Store = useProjectV1Store();
const databaseStore = useDatabaseV1Store();

const dataTableRows = computed(() => {
  const parentBranches = props.branches.filter((branch) => {
    return branch.parentBranch === "";
  });
  const parentRows: BranchRowData[] = parentBranches.map((branch) => {
    const [projectName] = getProjectAndSchemaDesignSheetId(branch.name);
    const project = projectV1Store.getProjectByName(
      `${projectNamePrefix}${projectName}`
    );
    const database = databaseStore.getDatabaseByName(branch.baselineDatabase);
    const baselineVersion = `(${database.effectiveEnvironmentEntity.title}) ${database.databaseName}`;

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
    return branch.parentBranch !== "";
  });
  for (const childBranch of childBranches) {
    const parentRow = parentRows.find(
      (row) => row.branch.name === childBranch.parentBranch
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

const getUpdatedTimeStr = (branch: Branch) => {
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

<style lang="postcss" scoped>
.bb-branch-data-table :deep(.n-data-table-expand-trigger) {
  @apply !w-5 !h-5 inline-flex justify-center items-center translate-y-0.5 rounded hover:bg-white hover:shadow;
}
.bb-branch-data-table :deep(.n-data-table-expand-trigger > .n-base-icon) {
  @apply !w-5 !h-5 flex flex-row justify-center items-center;
}
.bb-branch-data-table :deep(.n-data-table-td) {
  background-color: transparent !important;
}
</style>
