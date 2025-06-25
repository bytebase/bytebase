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
import { v4 as uuidv4 } from "uuid";
import { computed, ref } from "vue";
import { create } from "@bufbuild/protobuf";
import { PreviewTaskRunRollbackRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { useRouter } from "vue-router";
import {
  latestTaskRunForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { pushNotification, useSheetV1Store, useStorageStore, useCurrentProjectV1 } from "@/store";
import {
  extractIssueUID,
  extractProjectResourceName,
  hasProjectPermissionV2,
  sheetNameOfTaskV1,
} from "@/utils";

const router = useRouter();
const { issue, selectedTask } = useIssueContext();
const { project } = useCurrentProjectV1();

const isLoading = ref(false);

const allowRollback = computed((): boolean => {
  return hasProjectPermissionV2(project.value, "bb.issues.create");
});

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
  const request = create(PreviewTaskRunRollbackRequestSchema, {
    name: latestTaskRun.value.name,
  });
  const response = await rolloutServiceClientConnect.previewTaskRunRollback(request);
  const { statement } = response;
  isLoading.value = false;

  const sqlStorageKey = `bb.issues.sql.${uuidv4()}`;
  useStorageStore().put(sqlStorageKey, statement);
  const query: Record<string, any> = {
    template: "bb.issue.database.data.update",
    name: `Rollback ${selectedTask.value.target} in issue#${extractIssueUID(issue.value.name)}`,
    databaseList: selectedTask.value.target,
    description: `This issue is created to rollback the data of ${selectedTask.value.target} in issue #${extractIssueUID(issue.value.name)}`,
    sqlStorageKey,
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
