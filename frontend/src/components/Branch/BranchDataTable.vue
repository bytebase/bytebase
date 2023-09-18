<template>
  <NDataTable
    :loading="!ready"
    :default-expand-all="true"
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
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useUserStore } from "@/store";
import {
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";

type BranchRowData = {
  branch: SchemaDesign;
  name: string;
  branchName: string;
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

const dataTableRows = computed(() => {
  const parentBranches = props.branches.filter((branch) => {
    return branch.type === SchemaDesign_Type.MAIN_BRANCH;
  });
  const parentRows: BranchRowData[] = parentBranches.map((branch) => {
    return {
      branch: branch,
      name: branch.title,
      branchName: branch.title,
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
        branchName: childBranch.title,
        updatedTimeStr: getUpdatedTimeStr(childBranch),
      });
    }
  }

  return parentRows;
});

const dataTableColumns = computed(() => {
  return [
    {
      title: t("common.branch"),
      key: "branchName",
    },
    {
      title: t("common.updated"),
      key: "updatedTimeStr",
    },
  ];
});

const rowKey = (row: BranchRowData) => {
  return row.branch.name;
};

const rowProps = (row: BranchRowData) => {
  return {
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
