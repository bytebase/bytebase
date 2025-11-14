<template>
  <BBModal
    :title="overrideTitle ?? $t('task.check-result.title-general')"
    class="w-4xl!"
    header-class="whitespace-pre-wrap break-all gap-x-1"
    container-class="pt-0! -mt-px"
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
import { create as createProto } from "@bufbuild/protobuf";
import { NButton, NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import PlanCheckRunDetail from "@/components/PlanCheckRun/PlanCheckRunDetail.vue";
import { useProjectV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import {
  PlanCheckRun_Result_SqlReviewReportSchema,
  PlanCheckRun_ResultSchema,
  PlanCheckRun_Status,
  PlanCheckRunSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { type Advice, Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import type { Defer } from "@/utils";

const props = withDefaults(
  defineProps<{
    project: string;
    advices: Advice[];
    database?: ComposedDatabase;
    affectedRows?: bigint;
    riskLevel?: RiskLevel;
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
const projectStore = useProjectV1Store();

// disallow creating issues if advice statuses contains any error.
const restrictIssueCreationForSQLReview = computed((): boolean => {
  const project = projectStore.getProjectByName(props.project);
  return (
    (project?.enforceSqlReview || false) &&
    props.advices.some((advice) => advice.status === Advice_Level.ERROR)
  );
});

const riskLevelText = computed(() => {
  const { riskLevel } = props;
  if (riskLevel === RiskLevel.LOW) {
    return t("issue.risk-level.low");
  } else if (riskLevel === RiskLevel.MODERATE) {
    return t("issue.risk-level.moderate");
  } else if (riskLevel === RiskLevel.HIGH) {
    return t("issue.risk-level.high");
  }
  return "";
});

const planCheckRun = computed((): PlanCheckRun => {
  return createProto(PlanCheckRunSchema, {
    status: PlanCheckRun_Status.DONE,
    results: props.advices.map((advice) => {
      let status = Advice_Level.ADVICE_LEVEL_UNSPECIFIED;
      switch (advice.status) {
        case Advice_Level.SUCCESS:
          status = Advice_Level.SUCCESS;
          break;
        case Advice_Level.WARNING:
          status = Advice_Level.WARNING;
          break;
        case Advice_Level.ERROR:
          status = Advice_Level.ERROR;
          break;
      }
      return createProto(PlanCheckRun_ResultSchema, {
        status,
        title: advice.title,
        code: advice.code,
        content: advice.content,
        report: {
          case: "sqlReviewReport",
          value: createProto(PlanCheckRun_Result_SqlReviewReportSchema, {
            startPosition: advice.startPosition,
          }),
        },
      });
    }),
  });
});

const onClose = () => {
  props.confirm?.resolve(false);
  emit("close");
};
</script>
