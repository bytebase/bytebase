<template>
  <div class="md:flex md:items-center md:justify-between">
    <div class="flex-1 min-w-0">
      <div class="flex flex-col">
        <div class="flex items-center gap-x-2">
          <div v-if="!create">
            <IssueStatusIcon
              :issue-status="issue.status"
              :task-status="issueTaskStatus"
            />
          </div>
          <BBTextField
            class="my-px px-2 flex-1 text-lg font-bold truncate"
            :class="[create && '-ml-2']"
            :disabled="!allowEditNameAndDescription"
            :required="true"
            :focus-on-mount="create"
            :bordered="false"
            :value="state.name"
            :placeholder="'Issue name'"
            @end-editing="(text: string) => trySaveName(text)"
          />

          <div class="mt-4 flex space-x-3 md:mt-0 md:ml-4">
            <div v-if="showExportCenterLink">
              <router-link
                class="btn-primary"
                :to="{
                  name: 'workspace.export-center',
                  hash: `#${issue.id}`,
                }"
              >
                <heroicons-outline:download class="w-5 h-5 mr-2" />
                <span>{{ $t("export-center.self") }}</span>
              </router-link>
            </div>
            <div v-else-if="showSQLEditorLink">
              <button class="btn-primary" @click="gotoSQLEditor">
                <heroicons-solid:terminal class="w-5 h-5 mr-2" />
                <span>{{ $t("sql-editor.self") }}</span>
              </button>
            </div>
            <IssueReviewButtonGroup v-else-if="showReviewButtonGroup" />
            <CombinedRolloutButtonGroup v-else-if="showRolloutButtonGroup" />
          </div>
        </div>
        <div v-if="!create">
          <i18n-t
            keypath="issue.opened-by-at"
            tag="p"
            class="text-sm text-control-light"
          >
            <template #creator>
              <router-link
                :to="`/u/${issue.creator.id}`"
                class="font-medium text-control hover:underline"
                >{{ issue.creator.name }}</router-link
              >
            </template>
            <template #time>{{
              dayjs(issue.createdTs * 1000).format("LLL")
            }}</template>
          </i18n-t>
          <p
            v-if="pushEvent"
            class="mt-1 text-sm text-control-light flex flex-row items-center space-x-1"
          >
            <template v-if="pushEvent.vcsType.startsWith('GITLAB')">
              <img class="h-4 w-auto" src="../../assets/gitlab-logo.svg" />
            </template>
            <template v-else-if="pushEvent.vcsType.startsWith('GITHUB')">
              <img class="h-4 w-auto" src="../../assets/github-logo.svg" />
            </template>
            <template v-else-if="pushEvent.vcsType.startsWith('BITBUCKET')">
              <img class="h-4 w-auto" src="../../assets/bitbucket-logo.svg" />
            </template>
            <a :href="vcsBranchUrl" target="_blank" class="normal-link">{{
              `${vcsBranch}@${pushEvent.repositoryFullPath}`
            }}</a>

            <i18n-t
              v-if="commit && commit.id && commit.url"
              keypath="issue.commit-by-at"
              tag="span"
            >
              <template #id>
                <a :href="commit.url" target="_blank" class="normal-link"
                  >{{ commit.id.substring(0, 7) }}:</a
                >
              </template>
              <template #title>
                <span class="text-main">{{ commit.title }}</span>
              </template>
              <template #author>{{ pushEvent.authorName }}</template>
              <template #time>{{
                dayjs(commit.createdTs * 1000).format("LLL")
              }}</template>
            </i18n-t>
          </p>
        </div>
        <slot name="tips"></slot>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { reactive, watch, computed, Ref } from "vue";
import { useRouter } from "vue-router";
import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import { useCurrentUserV1, useDatabaseV1Store } from "@/store";
import {
  TaskDatabaseSchemaUpdatePayload,
  TaskDatabaseDataUpdatePayload,
  Issue,
  VCSPushEvent,
  PresetRoleType,
  GrantRequestPayload,
  UNKNOWN_ID,
} from "@/types";
import {
  activeTask,
  connectionV1Slug,
  extractUserUID,
  isDatabaseRelatedIssueType,
  isGrantRequestIssueType,
} from "@/utils";
import { convertFromCELString } from "@/utils/issue/cel";
import IssueStatusIcon from "./IssueStatusIcon.vue";
import { CombinedRolloutButtonGroup } from "./StatusTransitionButtonGroup";
import { useExtraIssueLogic, useIssueLogic } from "./logic";
import { IssueReviewButtonGroup } from "./review";

interface LocalState {
  editing: boolean;
  name: string;
}

const router = useRouter();
const currentUser = useCurrentUserV1();
const logic = useIssueLogic();
const create = logic.create;
const issue = logic.issue as Ref<Issue>;
const { allowEditNameAndDescription, updateName } = useExtraIssueLogic();
const issueReview = useIssueReviewContext();
const { done: reviewDone, error: reviewError } = issueReview;

const state = reactive<LocalState>({
  editing: false,
  name: issue.value.name,
});

const isFinishedGrantRequestIssueByCurrentUser = computed(() => {
  if (create.value) return false;
  if (issue.value.status !== "DONE") return false;
  if (!isGrantRequestIssueType(issue.value.type)) return false;
  if (
    String(issue.value.creator.id) !== extractUserUID(currentUser.value.name)
  ) {
    return false;
  }
  return true;
});

const showExportCenterLink = computed(() => {
  if (!isFinishedGrantRequestIssueByCurrentUser.value) return false;
  return issue.value.payload.grantRequest?.role === PresetRoleType.EXPORTER;
});

const showSQLEditorLink = computed(() => {
  if (!isFinishedGrantRequestIssueByCurrentUser.value) return false;
  return issue.value.payload.grantRequest?.role === PresetRoleType.QUERIER;
});

/**
 * Send back / Approve
 * + cancel issue (dropdown)
 */
const showReviewButtonGroup = computed(() => {
  if (create.value) return false;
  if (reviewError.value) return false;
  // User can cancel issue when it's in review.
  if (isGrantRequestIssueType(issue.value.type)) return true;
  return !reviewDone.value;
});

/**
 * Rollout / Retry
 * + cancel issue (dropdown)
 * + skip all failed tasks in current stage (dropdown)
 */
const showRolloutButtonGroup = computed(() => {
  if (create.value) return true;
  if (isGrantRequestIssueType(issue.value.type)) return false;

  return reviewDone.value;
});

const issueTaskStatus = computed(() => {
  // For grant request issue, we always show the status as "PENDING_APPROVAL" as task status.
  if (!isDatabaseRelatedIssueType(issue.value.type)) {
    return "PENDING_APPROVAL";
  }

  return activeTask(issue.value.pipeline!).status;
});

watch(
  () => issue.value,
  (curIssue) => {
    state.name = curIssue.name;
  }
);

const pushEvent = computed((): VCSPushEvent | undefined => {
  if (issue.value.type == "bb.issue.database.schema.update") {
    const payload = activeTask(issue.value.pipeline!)
      .payload as TaskDatabaseSchemaUpdatePayload;
    return payload?.pushEvent;
  } else if (issue.value.type == "bb.issue.database.data.update") {
    const payload = activeTask(issue.value.pipeline!)
      .payload as TaskDatabaseDataUpdatePayload;
    return payload?.pushEvent;
  }
  return undefined;
});

const commit = computed(() => {
  // Use commits[0] for new format
  // Use fileCommit for legacy data (if possible)
  // Use undefined otherwise
  return head(pushEvent.value?.commits) ?? pushEvent.value?.fileCommit;
});

const vcsBranch = computed((): string => {
  if (pushEvent.value) {
    return pushEvent.value.ref.replace(/^refs\/heads\//g, "");
  }
  return "";
});

const vcsBranchUrl = computed((): string => {
  if (pushEvent.value) {
    if (pushEvent.value.vcsType == "GITLAB") {
      return `${pushEvent.value.repositoryUrl}/-/tree/${vcsBranch.value}`;
    } else if (pushEvent.value.vcsType == "GITHUB") {
      return `${pushEvent.value.repositoryUrl}/tree/${vcsBranch.value}`;
    } else if (pushEvent.value.vcsType == "BITBUCKET") {
      return `${pushEvent.value.repositoryUrl}/src/${vcsBranch.value}`;
    }
  }
  return "";
});

const trySaveName = (text: string) => {
  state.name = text;
  if (text != issue.value.name) {
    updateName(state.name);
  }
};

const gotoSQLEditor = async () => {
  const grantRequest = issue.value.payload.grantRequest as GrantRequestPayload;
  const conditionExpression = await convertFromCELString(
    grantRequest.condition.expression
  );
  if (
    conditionExpression.databaseResources !== undefined &&
    conditionExpression.databaseResources.length > 0
  ) {
    const databaseResourceName = conditionExpression.databaseResources[0]
      .databaseName as string;
    const db = await useDatabaseV1Store().getOrFetchDatabaseByName(
      databaseResourceName
    );
    if (db.uid !== String(UNKNOWN_ID)) {
      const slug = connectionV1Slug(db.instanceEntity, db);
      const url = router.resolve({
        name: "sql-editor.detail",
        params: {
          connectionSlug: slug,
        },
      });
      window.open(url.fullPath, "__BLANK");
      return;
    }
  }
  const url = router.resolve({
    name: "sql-editor.home",
  });
  window.open(url.fullPath, "__BLANK");
};
</script>
