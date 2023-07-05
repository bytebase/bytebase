<template>
  <div class="md:flex md:items-center md:justify-between px-4 pb-4">
    <div class="flex-1 min-w-0">
      <div class="flex flex-col">
        <div class="flex items-center gap-x-2">
          <div v-if="!isCreating">
            <IssueStatusIcon
              :issue-status="issue.status"
              :task-status="issueTaskStatus"
            />
          </div>
          <BBTextField
            class="my-px px-2 flex-1 text-lg font-bold truncate"
            :class="[isCreating && '-ml-2']"
            :disabled="!allowEditNameAndDescription"
            :required="true"
            :focus-on-mount="isCreating"
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
                  hash: `#${issue.uid}`,
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

        <div class="flex items-center gap-x-4">
          <div class="flex items-center gap-x-1 text-sm">
            <div class="textlabel">{{ $t("common.project") }}</div>
            <div>-</div>
            <ProjectV1Name :project="project" />
          </div>

          <i18n-t
            v-if="!isCreating && creator"
            keypath="issue.opened-by-at"
            tag="div"
            class="text-sm text-control-light"
          >
            <template #creator>
              <router-link
                :to="`/u/${extractUserUID(creator.name)}`"
                class="font-medium text-control hover:underline"
                >{{ creator.title }}</router-link
              >
            </template>
            <template #time>{{
              dayjs(issue.createTime).format("LLL")
            }}</template>
          </i18n-t>
        </div>

        <div v-if="!isCreating">
          <!-- <p
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
          </p> -->
        </div>
        <slot name="tips"></slot>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, watch, computed } from "vue";

import IssueStatusIcon from "./IssueStatusIcon.vue";
import {
  activeTaskInRollout,
  extractUserResourceName,
  extractUserUID,
  isDatabaseRelatedIssue,
  isGrantRequestIssue,
} from "@/utils";
import { useCurrentUserV1, useUserStore } from "@/store";
import { useIssueContext } from "../logic";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { ProjectV1Name } from "@/components/v2";

interface LocalState {
  editing: boolean;
  name: string;
}

const currentUser = useCurrentUserV1();
const { isCreating, issue } = useIssueContext();

const state = reactive<LocalState>({
  editing: false,
  name: issue.value.title,
});

const creator = computed(() => {
  const email = extractUserResourceName(issue.value.creator);
  return useUserStore().getUserByEmail(email);
});

const project = computed(() => {
  return issue.value.projectEntity;
});

const isFinishedGrantRequestIssueByCurrentUser = computed(() => {
  if (isCreating.value) return false;
  if (issue.value.status !== IssueStatus.DONE) return false;
  if (!isGrantRequestIssue(issue.value)) return false;

  if (issue.value.creator !== currentUser.value.name) {
    return false;
  }
  return true;
});

const showExportCenterLink = computed(() => {
  if (!isFinishedGrantRequestIssueByCurrentUser.value) return false;
  return false; // todo
  // return issue.value.pa.payload.grantRequest?.role === PresetRoleType.EXPORTER;
});

const showSQLEditorLink = computed(() => {
  if (!isFinishedGrantRequestIssueByCurrentUser.value) return false;
  return false; // todo
  // return issue.value.payload.grantRequest?.role === PresetRoleType.QUERIER;
});

/**
 * Send back / Approve
 * + cancel issue (dropdown)
 */
const showReviewButtonGroup = computed(() => {
  if (isCreating.value) return false;
  // if (reviewError.value) return false;
  // User can cancel issue when it's in review.
  if (isGrantRequestIssue(issue.value)) return true;
  return false; // todo
  // return !reviewDone.value;
});

/**
 * Rollout / Retry
 * + cancel issue (dropdown)
 * + skip all failed tasks in current stage (dropdown)
 */
const showRolloutButtonGroup = computed(() => {
  if (isCreating.value) return true;
  if (isGrantRequestIssue(issue.value)) return false;

  return false; // todo
  // return reviewDone.value;
});

const issueTaskStatus = computed(() => {
  // For grant request issue, we always show the status as "PENDING_APPROVAL" as task status.
  if (!isDatabaseRelatedIssue(issue.value)) {
    return Task_Status.PENDING_APPROVAL;
  }

  return activeTaskInRollout(issue.value.rolloutEntity).status;
});

watch(
  () => issue.value,
  (curIssue) => {
    state.name = curIssue.title;
  }
);

const allowEditNameAndDescription = computed(() => {
  return false; // todo
});

// const pushEvent = computed((): VCSPushEvent | undefined => {
//   if (issue.value.type == "bb.issue.database.schema.update") {
//     const payload = activeTask(issue.value.pipeline!)
//       .payload as TaskDatabaseSchemaUpdatePayload;
//     return payload?.pushEvent;
//   } else if (issue.value.type == "bb.issue.database.data.update") {
//     const payload = activeTask(issue.value.pipeline!)
//       .payload as TaskDatabaseDataUpdatePayload;
//     return payload?.pushEvent;
//   }
//   return undefined;
// });

// const commit = computed(() => {
//   // Use commits[0] for new format
//   // Use fileCommit for legacy data (if possible)
//   // Use undefined otherwise
//   return head(pushEvent.value?.commits) ?? pushEvent.value?.fileCommit;
// });

// const vcsBranch = computed((): string => {
//   if (pushEvent.value) {
//     return pushEvent.value.ref.replace(/^refs\/heads\//g, "");
//   }
//   return "";
// });

// const vcsBranchUrl = computed((): string => {
//   if (pushEvent.value) {
//     if (pushEvent.value.vcsType == "GITLAB") {
//       return `${pushEvent.value.repositoryUrl}/-/tree/${vcsBranch.value}`;
//     } else if (pushEvent.value.vcsType == "GITHUB") {
//       return `${pushEvent.value.repositoryUrl}/tree/${vcsBranch.value}`;
//     } else if (pushEvent.value.vcsType == "BITBUCKET") {
//       return `${pushEvent.value.repositoryUrl}/src/${vcsBranch.value}`;
//     }
//   }
//   return "";
// });

const trySaveName = (text: string) => {
  state.name = text;
  if (text != issue.value.name) {
    // updateName(state.name);
  }
};

const gotoSQLEditor = async () => {
  // const grantRequest = issue.value.payload.grantRequest as GrantRequestPayload;
  // const conditionExpression = await convertFromCELString(
  //   grantRequest.condition.expression
  // );
  // if (
  //   conditionExpression.databaseResources !== undefined &&
  //   conditionExpression.databaseResources.length > 0
  // ) {
  //   const databaseResourceName = conditionExpression.databaseResources[0]
  //     .databaseName as string;
  //   const db = await useDatabaseV1Store().getOrFetchDatabaseByName(
  //     databaseResourceName
  //   );
  //   if (db.uid !== String(UNKNOWN_ID)) {
  //     const slug = connectionV1Slug(db.instanceEntity, db);
  //     const url = router.resolve({
  //       name: "sql-editor.detail",
  //       params: {
  //         connectionSlug: slug,
  //       },
  //     });
  //     window.open(url.fullPath, "__BLANK");
  //     return;
  //   }
  // }
  // const url = router.resolve({
  //   name: "sql-editor.home",
  // });
  // window.open(url.fullPath, "__BLANK");
};
</script>
