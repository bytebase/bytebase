<template>
  <NTooltip :disabled="issueCreateErrorList.length === 0" placement="top">
    <template #trigger>
      <NButton
        type="primary"
        size="medium"
        tag="div"
        :disabled="issueCreateErrorList.length > 0 || loading"
        :loading="loading"
        @click="handleCreateIssue"
      >
        {{ loading ? $t("common.creating") : $t("issue.create-issue") }}
      </NButton>
    </template>

    <template #default>
      <ErrorList :errors="issueCreateErrorList" />
    </template>
  </NTooltip>

  <!-- prevent clicking the page when creating in progress -->
  <div
    v-if="loading"
    v-zindexable="{ enabled: true }"
    class="fixed inset-0 pointer-events-auto flex flex-col items-center justify-center"
    @click.stop.prevent
  />
</template>

<script setup lang="ts">
import { uniqBy } from "lodash-es";
import { NTooltip, NButton, useDialog } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";
import { computed, nextTick, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { ErrorList } from "@/components/Plan/components/common";
import {
  planCheckRunListForSpec,
  usePlanContext,
} from "@/components/Plan/logic";
import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import { issueServiceClient, rolloutServiceClient } from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentUserV1, usePolicyV1Store } from "@/store";
import { emptyIssue, type ComposedIssue } from "@/types";
import { Issue, IssueStatus, Issue_Type } from "@/types/proto/v1/issue_service";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { PlanCheckRun_Result_Status } from "@/types/proto/v1/plan_service";
import {
  extractProjectResourceName,
  hasProjectPermissionV2,
  issueV1Slug,
} from "@/utils";

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const policyV1Store = usePolicyV1Store();
const { plan } = usePlanContext();
const loading = ref(false);
const restrictIssueCreationForSqlReviewPolicy = ref(false);

const planCheckStatus = computed((): PlanCheckRun_Result_Status => {
  const planCheckList = uniqBy(
    plan.value.steps.flatMap((step) =>
      step.specs.flatMap((spec) => planCheckRunListForSpec(plan.value, spec))
    ),
    (checkRun) => checkRun.name
  );
  const summary = planCheckRunSummaryForCheckRunList(planCheckList);
  if (summary.errorCount > 0) {
    return PlanCheckRun_Result_Status.ERROR;
  }
  if (summary.warnCount > 0) {
    return PlanCheckRun_Result_Status.WARNING;
  }
  return PlanCheckRun_Result_Status.SUCCESS;
});

const issueCreateErrorList = computed(() => {
  const errorList: string[] = [];
  if (!hasProjectPermissionV2(plan.value.projectEntity, "bb.plans.create")) {
    errorList.push(t("common.missing-required-permission"));
  }
  if (!plan.value.title.trim()) {
    errorList.push("Missing issue title");
  }
  if (
    planCheckStatus.value === PlanCheckRun_Result_Status.ERROR &&
    restrictIssueCreationForSqlReviewPolicy.value
  ) {
    errorList.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
      )
    );
  }
  return errorList;
});

watchEffect(async () => {
  const workspaceLevelPolicy =
    await policyV1Store.getOrFetchPolicyByParentAndType({
      parentPath: "",
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    });
  if (workspaceLevelPolicy?.restrictIssueCreationForSqlReviewPolicy?.disallow) {
    restrictIssueCreationForSqlReviewPolicy.value = true;
    return;
  }

  const projectLevelPolicy =
    await policyV1Store.getOrFetchPolicyByParentAndType({
      parentPath: plan.value.project,
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    });
  if (projectLevelPolicy?.restrictIssueCreationForSqlReviewPolicy?.disallow) {
    restrictIssueCreationForSqlReviewPolicy.value = true;
    return;
  }

  // Fall back to default value.
  restrictIssueCreationForSqlReviewPolicy.value = false;
});

const handleCreateIssue = async () => {
  dialog.info({
    title: t("common.confirm"),
    content: t("issue.this-plan-will-be-converted-to-a-new-issue"),
    negativeText: t("common.cancel"),
    positiveText: t("common.create"),
    onPositiveClick: async () => {
      await doCreateIssue();
    },
  });
};

const doCreateIssue = async () => {
  loading.value = true;
  // TODO(steven): Check plan check results before creating issue.

  try {
    const createdIssue = await issueServiceClient.createIssue({
      parent: plan.value.project,
      issue: {
        ...Issue.fromPartial(buildIssue()),
        rollout: "",
        plan: plan.value.name,
      },
    });
    const composedIssue: ComposedIssue = {
      ...emptyIssue(),
      ...createdIssue,
      planEntity: plan.value,
    };
    const createdRollout = await rolloutServiceClient.createRollout({
      parent: plan.value.project,
      rollout: {
        plan: plan.value.name,
      },
    });

    composedIssue.rollout = createdRollout.name;
    composedIssue.rolloutEntity = createdRollout;

    nextTick(() => {
      router.push({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(plan.value.project),
          issueSlug: issueV1Slug(composedIssue),
        },
      });
    });

    return composedIssue;
  } catch {
    loading.value = false;
  }
};

const buildIssue = () => {
  const issue = emptyIssue();
  const me = useCurrentUserV1();
  issue.creator = `users/${me.value.email}`;
  issue.creatorEntity = me.value;
  issue.project = plan.value.projectEntity.name;
  issue.projectEntity = plan.value.projectEntity;
  issue.title = plan.value.title;
  issue.description = plan.value.description;
  issue.status = IssueStatus.OPEN;
  issue.type = Issue_Type.DATABASE_CHANGE;
  return issue;
};
</script>
