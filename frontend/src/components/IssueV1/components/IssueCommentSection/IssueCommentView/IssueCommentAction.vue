<template>
  <div class="ml-3 min-w-0 flex-1">
    <div class="min-w-0 flex-1 pt-1 flex justify-between">
      <div class="text-sm text-control-light space-x-1">
        <ActionCreator
          v-if="
            extractUserResourceName(issueComment.creator) !==
              userStore.systemBotUser?.email ||
            issueComment.type === IssueCommentType.USER_COMMENT
          "
          :creator="issueComment.creator"
        />

        <ActionSentence :issue="issue" :issue-comment="issueComment" />

        <NButton
          v-if="showRestoreButton(issueComment)"
          size="small"
          @click.prevent="createRestoreIssue(issueComment)"
        >
          <span>{{ $t("activity.restore") }}</span>
        </NButton>

        <HumanizeTs
          :ts="(issueComment.createTime?.getTime() ?? 0) / 1000"
          class="ml-1"
        />

        <span
          v-if="
            issueComment.createTime?.getTime() !==
              issueComment.updateTime?.getTime() &&
            issueComment.type === IssueCommentType.USER_COMMENT
          "
        >
          <span>({{ $t("common.edited") }}</span>
          <HumanizeTs
            :ts="(issueComment.updateTime?.getTime() ?? 0) / 1000"
            class="ml-1"
          />)
        </span>

        <span
          v-if="similar.length > 0"
          class="text-sm font-normal text-gray-400 ml-1"
        >
          {{
            $t("activity.n-similar-activities", {
              count: similar.length + 1,
            })
          }}
        </span>
      </div>

      <slot name="subject-suffix"></slot>
    </div>
    <div class="mt-2 text-sm text-control whitespace-pre-wrap">
      <slot name="comment" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useIssueContext, databaseForTask } from "@/components/IssueV1/logic";
import HumanizeTs from "@/components/misc/HumanizeTs.vue";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  IssueCommentType,
  useSQLStore,
  type ComposedIssueComment,
} from "@/store";
import { useSheetV1Store, useUserStore } from "@/store";
import type { ComposedIssue } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { IssueComment_TaskPriorBackup } from "@/types/proto/v1/issue_service";
import type { IssueComment } from "@/types/proto/v1/issue_service";
import {
  extractUserResourceName,
  extractProjectResourceName,
  sheetNameOfTaskV1,
  extractDatabaseResourceName,
} from "@/utils";
import ActionCreator from "./ActionCreator.vue";
import ActionSentence from "./ActionSentence.vue";

const props = defineProps<{
  issue: ComposedIssue;
  index: number;
  issueComment: ComposedIssueComment;
  similar: ComposedIssueComment[];
}>();

const { selectedTask } = useIssueContext();
const router = useRouter();
const userStore = useUserStore();

const coreDatabaseInfo = computed(() => {
  return databaseForTask(props.issue, selectedTask.value);
});

const showRestoreButton = (comment: ComposedIssueComment) => {
  if (comment.type !== IssueCommentType.TASK_PRIOR_BACKUP) {
    return false;
  }
  if (!selectedTask.value) {
    return false;
  }
  if (coreDatabaseInfo.value.instanceResource.engine !== Engine.MYSQL) {
    return false;
  }
  return true;
};

const createRestoreIssue = async (comment: IssueComment) => {
  const {
    tables,
    originalLine,
    database: backupDatabase,
  } = IssueComment_TaskPriorBackup.fromPartial(comment.taskPriorBackup || {});

  const issueNameParts: string[] = [];
  issueNameParts.push(
    `Restore the sql at line ${originalLine} of (${props.issue.title})`
  );
  const datetime = dayjs().format("%MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);
  const sheetName = sheetNameOfTaskV1(selectedTask.value);
  const sheet = useSheetV1Store().getSheetByName(sheetName);
  if (!sheet) {
    console.error(`Sheet ${sheetName} not found`);
    return;
  }

  const { instance } = extractDatabaseResourceName(selectedTask.value.target);
  const { statement: restoreSQL } = await useSQLStore().generateRestoreSQL({
    name: selectedTask.value.target,
    sheet: sheet.name,
    backupDataSource: `${instance}/databases/${backupDatabase.length > 0 ? backupDatabase : "bbdataarchive"}`,
    backupTable: tables[0].table,
  });
  if (!restoreSQL) {
    console.error("Failed to generate restore SQL");
    return;
  }

  const query: Record<string, any> = {
    template: "bb.issue.database.data.update",
    name: issueNameParts.join(" "),
    databaseList: selectedTask.value.target,
    sql: restoreSQL,
  };

  router.push({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(selectedTask.value.name),
      issueSlug: "create",
    },
    query,
  });
};
</script>
