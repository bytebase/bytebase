<template>
  <BBModal
    :title="overrideTitle ?? $t('task.check-result.title-general')"
    class="!w-[56rem]"
    header-class="whitespace-pre-wrap break-all gap-x-1"
    container-class="!pt-0 -mt-px"
    mask-closable
    @close="onClose"
  >
    <div class="w-full flex flex-row gap-2 mb-2">
      <NTag round v-if="riskLevelText">
        <span class="text-gray-400 text-sm mr-1">{{
          $t("issue.risk-level.self")
        }}</span>
        <span class="text-sm font-medium">
          {{ riskLevelText }}
        </span>
      </NTag>
      <NTag round v-if="riskLevelText || (affectedRows && affectedRows > 0)">
        <span class="text-gray-400 text-sm mr-1">{{
          $t("task.check-type.affected-rows.self")
        }}</span>
        <span class="text-sm font-medium">
          {{ affectedRows }}
        </span>
      </NTag>
    </div>
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
      <NButton @click="onClose">
        {{ $t("issue.sql-check.back-to-edit") }}
      </NButton>
      <NButton
        v-if="allowForceContinue && !restrictIssueCreationForSQLReview"
        type="primary"
        @click="confirm!.resolve(true)"
      >
        {{ $t("common.continue-anyway") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { NButton, NTag } from "naive-ui";
import { computed, watchEffect, ref } from "vue";
import { useI18n } from "vue-i18n";
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
import { CheckReleaseResponse_RiskLevel } from "@/types/proto/v1/release_service";
import type { Advice } from "@/types/proto/v1/sql_service";
import { Advice_Status } from "@/types/proto/v1/sql_service";
import type { Defer } from "@/utils";

const props = withDefaults(
  defineProps<{
    project: string;
    advices: Advice[];
    database?: ComposedDatabase;
    affectedRows?: number;
    riskLevel?: CheckReleaseResponse_RiskLevel;
    overrideTitle?: string;
    confirm?: Defer<boolean>;
    showCodeLocation?: boolean;
    allowForceContinue?: boolean;
  }>(),
  {
    showCodeLocation: false,
    allowForceContinue: true,
    overrideTitle: undefined,
    confirm: undefined,
  }
);

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const restrictIssueCreationForSqlReviewPolicy = ref(false);
const policyV1Store = usePolicyV1Store();

// disallow creating issues if advice statuses contains any error.
const restrictIssueCreationForSQLReview = computed((): boolean => {
  return (
    restrictIssueCreationForSqlReviewPolicy.value &&
    props.advices.some((advice) => advice.status === Advice_Status.ERROR)
  );
});

const riskLevelText = computed(() => {
  const { riskLevel } = props;
  if (riskLevel === CheckReleaseResponse_RiskLevel.LOW) {
    return t("issue.risk-level.low");
  } else if (riskLevel === CheckReleaseResponse_RiskLevel.MODERATE) {
    return t("issue.risk-level.moderate");
  } else if (riskLevel === CheckReleaseResponse_RiskLevel.HIGH) {
    return t("issue.risk-level.high");
  }
  return "";
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
      parentPath: props.project,
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
          line: advice.startPosition?.line ?? 0,
          column: advice.startPosition?.column ?? Number.MAX_SAFE_INTEGER,
        }),
      });
    }),
  });
});

const onClose = () => {
  props.confirm?.resolve(false);
  emit("close");
};
</script>
