<template>
  <NButton size="small" :loading="isLoading" @click="createRestoreIssue">
    <template #icon>
      <Undo2Icon class="w-4 h-auto" />
    </template>
    {{ $t("common.rollback") }}
  </NButton>
</template>

<script lang="ts" setup>
import { Undo2Icon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import {
  latestTaskRunForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { pushNotification, useSheetV1Store, useSQLStore } from "@/store";
import { TaskRun_PriorBackupDetail_Item_Table } from "@/types/proto/v1/rollout_service";
import { extractProjectResourceName, sheetNameOfTaskV1 } from "@/utils";
import { usePreBackupContext } from "./common";

const router = useRouter();
const { issue, selectedTask } = useIssueContext();
const { allowRollback } = usePreBackupContext();

const isLoading = ref(false);

const latestTaskRun = computed(() =>
  latestTaskRunForTask(issue.value, selectedTask.value)
);

const createRestoreIssue = async () => {
  if (!allowRollback.value) {
    return;
  }
  if (!latestTaskRun.value?.priorBackupDetail) {
    return;
  }

  const sheetName = sheetNameOfTaskV1(selectedTask.value);
  const sheet = useSheetV1Store().getSheetByName(sheetName);
  if (!sheet) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Sheet ${sheetName} not found`,
    });
    return;
  }

  isLoading.value = true;
  const statements: string[] = [];
  for (const item of latestTaskRun.value.priorBackupDetail.items) {
    const targetTable = TaskRun_PriorBackupDetail_Item_Table.fromPartial({
      ...item.targetTable,
    });
    const { statement } = await useSQLStore().generateRestoreSQL({
      name: selectedTask.value.target,
      sheet: sheet.name,
      backupDataSource: targetTable.database,
      backupTable: targetTable.table,
    });
    statements.push(statement);
  }
  isLoading.value = false;

  const query: Record<string, any> = {
    template: "bb.issue.database.data.update",
    name: `Rollback ${selectedTask.value.title} in issue#${issue.value.uid}`,
    databaseList: selectedTask.value.target,
    sql: statements.join("\n"),
    description: `This issue is created to rollback the data of ${selectedTask.value.title} in issue#${issue.value.uid}`,
  };
  router.push({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(issue.value.name),
      issueSlug: "create",
    },
    query,
  });
};
</script>
