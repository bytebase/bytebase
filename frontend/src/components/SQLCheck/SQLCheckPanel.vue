<template>
  <BBModal
    :title="overrideTitle ?? $t('task.check-result.title-general')"
    :show-close="!confirm"
    :close-on-esc="!confirm"
    class="!w-[56rem]"
    header-class="whitespace-pre-wrap break-all gap-x-1"
    container-class="!pt-0 -mt-px"
    @close="$emit('close')"
  >
    <PlanCheckRunDetail
      :plan-check-run="planCheckRun"
      :database="database"
      :show-code-location="showCodeLocation"
    >
      <template #row-title-extra="{ row }">
        <slot name="row-title-extra" :row="row" />
      </template>
    </PlanCheckRunDetail>

    <div
      v-if="confirm"
      class="flex flex-row justify-end items-center gap-x-3 mt-3"
    >
      <NButton @click="confirm!.resolve(false)">
        {{ $t("issue.sql-check.back-to-edit") }}
      </NButton>
      <NButton
        v-if="allowForceContinue && !restrictIssueCreationForSQLReview"
        type="primary"
        @click="confirm!.resolve(true)"
      >
        {{ $t("issue.sql-check.continue-anyway") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, watchEffect, ref } from "vue";
import { BBModal } from "@/bbkit";
import PlanCheckRunDetail from "@/components/PlanCheckRun/PlanCheckRunDetail.vue";
import { usePolicyV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import {
  PlanCheckRun,
  PlanCheckRun_Result,
  PlanCheckRun_Result_Status,
  PlanCheckRun_Result_SqlReviewReport,
  PlanCheckRun_Status,
} from "@/types/proto/v1/plan_service";
import type { Advice } from "@/types/proto/v1/sql_service";
import { Advice_Status } from "@/types/proto/v1/sql_service";
import type { Defer } from "@/utils";

const props = withDefaults(
  defineProps<{
    database: ComposedDatabase;
    advices: Advice[];
    overrideTitle?: string;
    confirm?: Defer<boolean>;
    showCodeLocation?: boolean;
    allowForceContinue?: boolean;
  }>(),
  {
    allowForceContinue: true,
  }
);

defineEmits<{
  (event: "close"): void;
}>();

const restrictIssueCreationForSqlReviewPolicy = ref(false);
const policyV1Store = usePolicyV1Store();

// disallow creating issues if advice statuses contains any error.
const restrictIssueCreationForSQLReview = computed((): boolean => {
  return (
    restrictIssueCreationForSqlReviewPolicy.value &&
    props.advices.some((advice) => advice.status === Advice_Status.ERROR)
  );
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
      parentPath: props.database.project,
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    });
  if (projectLevelPolicy?.restrictIssueCreationForSqlReviewPolicy?.disallow) {
    restrictIssueCreationForSqlReviewPolicy.value = true;
    return;
  }

  // Fall back to default value.
  restrictIssueCreationForSqlReviewPolicy.value = false;
});

const planCheckRun = computed((): PlanCheckRun => {
  return PlanCheckRun.fromPartial({
    status: PlanCheckRun_Status.DONE,
    results: props.advices.map((advice) => {
      let status = PlanCheckRun_Result_Status.STATUS_UNSPECIFIED;
      switch (advice.status) {
        case Advice_Status.SUCCESS:
          status = PlanCheckRun_Result_Status.SUCCESS;
          break;
        case Advice_Status.WARNING:
          status = PlanCheckRun_Result_Status.WARNING;
          break;
        case Advice_Status.ERROR:
          status = PlanCheckRun_Result_Status.ERROR;
          break;
      }
      return PlanCheckRun_Result.fromPartial({
        status,
        title: advice.title,
        code: advice.code,
        content: advice.content,
        sqlReviewReport: PlanCheckRun_Result_SqlReviewReport.fromPartial({
          line: advice.line,
          column: advice.column,
          detail: advice.detail,
          code: advice.code,
        }),
      });
    }),
  });
});
</script>
