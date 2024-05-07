<template>
  <div class="text-sm font-medium">
    <template v-if="!config?.rollbackEnabled">
      <LogButton />
    </template>
    <template v-else>
      <template v-if="config?.rollbackSqlStatus === RollbackSqlStatus.FAILED">
        <NTooltip>
          <template #trigger>
            <span class="text-error">{{ $t("task.rollback.failed") }}</span>
          </template>

          <div v-if="config.rollbackError" class="max-w-[20rem]">
            {{ config.rollbackError }}
          </div>
        </NTooltip>
      </template>
      <template
        v-else-if="config?.rollbackSqlStatus === RollbackSqlStatus.DONE"
      >
        <NButton
          v-if="allowPreviewRollback"
          size="small"
          @click="tryRollbackTask"
        >
          {{ $t("task.rollback.preview-rollback-issue") }}
        </NButton>
        <NTooltip v-else :disabled="!!config?.rollbackSheet">
          <template #trigger>
            <NButton :disabled="true" size="small" tag="div">
              {{ $t("task.rollback.preview-rollback-issue") }}
            </NButton>
          </template>

          <div v-if="!config?.rollbackSheet" class="whitespace-pre-line">
            {{ $t("task.rollback.empty-rollback-statement") }}
            <HideInStandaloneMode>
              <LearnMoreLink
                url="https://www.bytebase.com/docs/change-database/rollback-data-changes?source=console#why-i-get-the-rollback-sheet-is-empty"
                color="light"
                class="ml-1"
              />
            </HideInStandaloneMode>
          </div>
        </NTooltip>
      </template>
      <template v-else>
        <LoggingButton />
      </template>
    </template>
  </div>
</template>
<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import { databaseForTask, useIssueContext } from "@/components/IssueV1";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useSheetV1Store } from "@/store";
import { Task_DatabaseDataUpdate_RollbackSqlStatus as RollbackSqlStatus } from "@/types/proto/v1/rollout_service";
import { extractProjectResourceName, extractSheetUID } from "@/utils";
import LogButton from "./LogButton.vue";
import LoggingButton from "./LoggingButton.vue";
import { useRollbackContext } from "./common";

type LocalState = {
  loading: boolean;
};

const router = useRouter();
const sheetV1Store = useSheetV1Store();

const { issue, selectedTask: task } = useIssueContext();
const { allowRollback } = useRollbackContext();

const config = computed(() => task.value.databaseDataUpdate);

const state = reactive<LocalState>({
  loading: false,
});

const allowPreviewRollback = computed(() => {
  if (!allowRollback.value) {
    return false;
  }
  if (!config.value?.rollbackSheet) {
    return false;
  }
  return true;
});

const tryRollbackTask = async () => {
  // const navigateToRollbackIssue = () => {
  //   if (!state.rollbackIssue) return;
  //   router.push(`/issue/${state.rollbackIssue.uid}`);
  // };

  // if (state.rollbackIssue) {
  //   return navigateToRollbackIssue();
  // }

  if (!config.value) return;
  try {
    state.loading = true;

    const issueName = [
      `Rollback`,
      `#${issue.value.uid}`,
      `${issue.value.title}`,
    ].join(" ");

    const originalSheet = await sheetV1Store.getOrFetchSheetByName(
      config.value.sheet,
      "FULL"
    );
    const description = [
      "The original SQL statement:",
      `${new TextDecoder().decode(originalSheet?.content)}`,
    ].join("\n");

    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(issue.value.project),
        issueSlug: "create",
      },
      query: {
        template: "bb.issue.database.data.update",
        name: issueName,
        databaseList: databaseForTask(issue.value, task.value).name,
        rollbackIssueId: issue.value.uid,
        rollbackTaskIdList: [task.value.uid].join(","),
        sheetId: extractSheetUID(config.value.rollbackSheet),
        description,
      },
    });
  } finally {
    state.loading = false;
  }
};
</script>
