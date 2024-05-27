<template>
  <div class="step" :class="stepClass">
    <div class="text" @click="handleClickStep">
      <div class="text-sm min-w-32 lg:min-w-fit space-x-1 whitespace-nowrap">
        <heroicons:arrow-small-right
          v-if="isSelectedStep"
          class="w-5 h-5 inline-block mb-0.5"
        />
        <span>{{ step.title }} {{ $t("common.stage") }}</span>
      </div>
    </div>

    <NTooltip
      v-if="isCreating && !validateSpecs()"
      trigger="hover"
      placement="top"
    >
      <template #trigger>
        <heroicons:exclamation-circle-solid
          class="w-6 h-6 ml-2 text-error hover:text-error-hover"
        />
      </template>
      <span>{{ $t("issue.missing-sql-statement") }}</span>
    </NTooltip>
    <NTooltip
      v-if="
        !isCreating &&
        (planCheckStatus === PlanCheckRun_Result_Status.ERROR ||
          planCheckStatus === PlanCheckRun_Result_Status.WARNING)
      "
      trigger="hover"
      placement="top"
    >
      <template #trigger>
        <heroicons:exclamation-circle-solid
          class="w-6 h-6 ml-2"
          :class="[
            planCheckStatus === PlanCheckRun_Result_Status.ERROR
              ? 'text-error hover:text-error-hover'
              : 'text-warning hover:text-warning-hover',
          ]"
        />
      </template>
      <span>{{
        $t(
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
        )
      }}</span>
    </NTooltip>
  </div>
</template>

<script lang="ts" setup>
import { head, isEqual, uniqBy } from "lodash-es";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import {
  isValidSpec,
  planCheckRunListForSpec,
  planCheckRunSummaryForCheckRunList,
  usePlanContext,
} from "@/components/Plan/logic";
import type { Plan_Step } from "@/types/proto/v1/plan_service";
import { PlanCheckRun_Result_Status } from "@/types/proto/v1/plan_service";

const props = defineProps<{
  step: Plan_Step;
}>();

const { isCreating, plan, selectedStep, events } = usePlanContext();

const isSelectedStep = computed(() => {
  return isEqual(props.step, selectedStep.value);
});

const stepClass = computed(() => {
  const classList: string[] = [];
  if (isSelectedStep.value) classList.push("selected");
  return classList;
});

const validateSpecs = () => {
  return props.step.specs.every((spec) => isValidSpec(spec));
};

const planCheckStatus = computed((): PlanCheckRun_Result_Status => {
  if (isCreating.value) return PlanCheckRun_Result_Status.UNRECOGNIZED;
  const planCheckList = uniqBy(
    props.step.specs.flatMap((spec) =>
      planCheckRunListForSpec(plan.value, spec)
    ),
    (checkRun) => checkRun.uid
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

const handleClickStep = () => {
  if (isSelectedStep.value) return;

  const spec = head(props.step.specs);
  if (!spec) {
    return;
  }
  events.emit("select-spec", { spec });
};
</script>

<style scoped lang="postcss">
.step {
  @apply cursor-default flex items-center justify-start w-full text-sm relative;
  @apply lg:flex-1;
}
.step .text {
  @apply cursor-pointer flex-1 flex flex-col text-gray-600 font-normal;
}
.step.selected .text {
  @apply font-bold text-accent underline;
}
</style>
