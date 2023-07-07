<template>
  <div class="flex items-center gap-x-3">
    <div class="issue-debug">
      <div>showReviewButtonGroup: {{ showReviewButtonGroup }}</div>
      <div>showRolloutButtonGroup: {{ showRolloutButtonGroup }}</div>
      <div>
        isFinishedGrantRequestIssueByCurrentUser:
        {{ isFinishedGrantRequestIssueByCurrentUser }}
      </div>
      <div>showExportCenterLink: {{ showExportCenterLink }}</div>
      <div>showSQLEditorLink: {{ showSQLEditorLink }}</div>
    </div>
    <template v-if="false">
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
    </template>
  </div>
</template>

<script setup lang="ts">
import { useCurrentUserV1 } from "@/store";
import { useIssueContext } from "../../logic";
import { computed } from "vue";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { isGrantRequestIssue } from "@/utils";

const currentUser = useCurrentUserV1();
const { isCreating, issue, reviewContext } = useIssueContext();
const { done: reviewDone } = reviewContext;

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
  return !reviewDone.value;
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
