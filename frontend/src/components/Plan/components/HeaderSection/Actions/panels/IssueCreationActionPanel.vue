<template>
  <CommonDrawer
    :show="action !== undefined"
    :title="title"
    :loading="state.loading"
    @close="$emit('close')"
  >
    <template #default>
      <div v-if="action" class="flex flex-col gap-y-4 px-1">
        <div class="flex flex-col gap-y-1">
          <div class="font-medium text-control">{{ $t("plan.self") }}</div>
          <div class="textinfolabel">
            {{ plan.title }}
          </div>
        </div>

        <div class="flex flex-col gap-y-1">
          <div class="font-medium text-control">
            {{ $t("common.description") }}
          </div>
          <div class="textinfolabel">
            {{ plan.description || "No description" }}
          </div>
        </div>
      </div>
    </template>
    <template #footer>
      <div
        v-if="action"
        class="w-full flex flex-row justify-end items-center gap-2"
      >
        <div class="flex justify-end gap-x-3">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>

          <NTooltip :disabled="confirmErrors.length === 0" placement="top">
            <template #trigger>
              <NButton
                type="primary"
                :disabled="confirmErrors.length > 0"
                @click="handleConfirm"
              >
                {{ $t("common.create") }}
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="confirmErrors" />
            </template>
          </NTooltip>
        </div>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { uniqBy } from "lodash-es";
import { NButton, NTooltip } from "naive-ui";
import { computed, nextTick, reactive, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { ErrorList } from "@/components/Plan/components/common";
import {
  planCheckRunListForSpec,
  usePlanContext,
} from "@/components/Plan/logic";
import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import {
  issueServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
} from "@/router/dashboard/projectV1";
import {
  useCurrentProjectV1,
  useCurrentUserV1,
  usePolicyV1Store,
} from "@/store";
import { emptyIssue } from "@/types";
import { CreateIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { Issue, IssueStatus, Issue_Type } from "@/types/proto/v1/issue_service";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanCheckRun_Result_Status } from "@/types/proto/v1/plan_service";
import {
  extractProjectResourceName,
  hasProjectPermissionV2,
  extractIssueUID,
  isDev,
  issueV1Slug,
} from "@/utils";
import {
  convertNewIssueToOld,
  convertOldIssueToNew,
} from "@/utils/v1/issue-conversions";

type IssueCreationAction = "CREATE";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action?: IssueCreationAction;
}>();
const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  loading: false,
});
const { project } = useCurrentProjectV1();
const currentUser = useCurrentUserV1();
const policyV1Store = usePolicyV1Store();
const { plan, planCheckRunList, events } = usePlanContext();
const restrictIssueCreationForSqlReviewPolicy = ref(false);

const title = computed(() => {
  switch (props.action) {
    case "CREATE":
      return t("plan.ready-for-review");
  }
  return ""; // Make linter happy
});

const planCheckStatus = computed((): PlanCheckRun_Result_Status => {
  const planCheckList = uniqBy(
    plan.value.specs.flatMap((spec) =>
      planCheckRunListForSpec(planCheckRunList.value, spec)
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

const confirmErrors = computed(() => {
  const errors: string[] = [];

  if (!hasProjectPermissionV2(project.value, "bb.plans.create")) {
    errors.push(t("common.missing-required-permission"));
  }

  if (!plan.value.title.trim()) {
    errors.push("Missing issue title");
  }

  if (
    planCheckStatus.value === PlanCheckRun_Result_Status.ERROR &&
    restrictIssueCreationForSqlReviewPolicy.value
  ) {
    errors.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
      )
    );
  }

  return errors;
});

watchEffect(async () => {
  const workspaceLevelPolicy =
    await policyV1Store.getOrFetchPolicyByParentAndType({
      parentPath: "",
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    });
  if (workspaceLevelPolicy?.policy?.case === "restrictIssueCreationForSqlReviewPolicy" &&
      workspaceLevelPolicy.policy.value.disallow) {
    restrictIssueCreationForSqlReviewPolicy.value = true;
    return;
  }

  const projectLevelPolicy =
    await policyV1Store.getOrFetchPolicyByParentAndType({
      parentPath: project.value.name,
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    });
  if (projectLevelPolicy?.policy?.case === "restrictIssueCreationForSqlReviewPolicy" &&
      projectLevelPolicy.policy.value.disallow) {
    restrictIssueCreationForSqlReviewPolicy.value = true;
    return;
  }

  // Fall back to default value.
  restrictIssueCreationForSqlReviewPolicy.value = false;
});

const handleConfirm = async () => {
  await doCreateIssue();
};

const doCreateIssue = async () => {
  state.loading = true;

  try {
    const issueToCreate = {
      ...Issue.fromPartial(buildIssue()),
      rollout: "",
      plan: plan.value.name,
    };
    const newIssue = convertOldIssueToNew(issueToCreate);
    const issueRequest = create(CreateIssueRequestSchema, {
      parent: project.value.name,
      issue: newIssue,
    });
    const createdIssue =
      await issueServiceClientConnect.createIssue(issueRequest);
    convertNewIssueToOld(createdIssue);

    // Then create the rollout from the plan
    const rolloutRequest = create(CreateRolloutRequestSchema, {
      parent: project.value.name,
      rollout: {
        title: plan.value.title,
        plan: plan.value.name,
      },
    });
    await rolloutServiceClientConnect.createRollout(rolloutRequest);

    // Emit status changed to refresh the UI
    events.emit("status-changed", { eager: true });

    nextTick(() => {
      if (isDev()) {
        router.push({
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
          params: {
            projectId: extractProjectResourceName(plan.value.name),
            issueId: extractIssueUID(createdIssue.name),
          },
        });
      } else {
        // TODO(steven): remove me please.
        router.push({
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
          params: {
            projectId: extractProjectResourceName(plan.value.name),
            issueSlug: issueV1Slug(createdIssue.name, createdIssue.title),
          },
        });
      }
    });
  } finally {
    state.loading = false;
    emit("close");
  }
};

const buildIssue = () => {
  const issue = emptyIssue();
  issue.creator = `users/${currentUser.value.email}`;
  issue.title = plan.value.title;
  issue.description = plan.value.description;
  issue.status = IssueStatus.OPEN;
  issue.type = Issue_Type.DATABASE_CHANGE;
  return issue;
};
</script>
