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
    <PlanCheckDetail
      :plan-check-run="planCheckRun"
      :environment="environment"
    />

    <div
      v-if="confirm"
      class="flex flex-row justify-end items-center gap-x-3 mt-4"
    >
      <NButton @click="confirm.resolve(false)">
        {{ $t("issue.sql-check.back-to-edit") }}
      </NButton>
      <NButton
        v-if="!restrictIssueCreationForSQLReview"
        type="primary"
        @click="confirm.resolve(true)"
      >
        {{ $t("issue.sql-check.continue-anyway") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, onMounted } from "vue";
import { usePolicyV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import {
  PlanCheckRun,
  PlanCheckRun_Result,
  PlanCheckRun_Result_Status,
  PlanCheckRun_Result_SqlReviewReport,
  PlanCheckRun_Status,
} from "@/types/proto/v1/rollout_service";
import { Advice, Advice_Status } from "@/types/proto/v1/sql_service";
import { Defer } from "@/utils";

const { advices, database } = defineProps<{
  database: ComposedDatabase;
  advices: Advice[];
  overrideTitle?: string;
  confirm?: Defer<boolean>;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const policyV1Store = usePolicyV1Store();

// disallow creating issues if advice statuses contains any error.
const restrictIssueCreationForSQLReview = computed((): boolean => {
  return (
    (policyV1Store.getPolicyByName(
      "policies/RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW"
    )?.restrictIssueCreationForSqlReviewPolicy?.disallow ??
      false) &&
    advices.some((advice) => advice.status === Advice_Status.ERROR)
  );
});

onMounted(async () => {
  await prepareOrgPolicy();
});

const prepareOrgPolicy = async () => {
  await policyV1Store.getOrFetchPolicyByName(
    "policies/RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW"
  );
};

const environment = computed(() => {
  return database.effectiveEnvironmentEntity.name;
});

const planCheckRun = computed((): PlanCheckRun => {
  return PlanCheckRun.fromPartial({
    status: PlanCheckRun_Status.DONE,
    results: advices.map((advice) => {
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
