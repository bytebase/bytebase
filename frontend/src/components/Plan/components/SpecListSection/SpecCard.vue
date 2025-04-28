<template>
  <div
    class="px-3 py-2 w-full cursor-pointer border rounded lg:flex-1 flex justify-start items-center overflow-hidden gap-x-1"
    :class="specClass"
    @click="onClickSpec(spec)"
  >
    <NTooltip
      v-if="
        [
          PlanCheckRun_Result_Status.WARNING,
          PlanCheckRun_Result_Status.ERROR,
        ].includes(planCheckStatus)
      "
      trigger="hover"
      placement="top"
    >
      <template #trigger>
        <heroicons:exclamation-circle-solid
          class="w-5 h-5 shrink-0"
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
    <div
      v-if="isDatabaseChangeSpec(spec)"
      class="flex items-center gap-2 truncate"
    >
      <InstanceV1Name
        :instance="databaseForSpec(plan, spec).instanceResource"
        :link="false"
        class="text-gray-500 text-sm"
      />
      <span class="truncate text-sm">{{
        databaseForSpec(plan, spec).databaseName
      }}</span>
    </div>
    <!-- Fallback to show the database group name if the spec is a grouping change spec in old plans. -->
    <div
      v-else-if="isGroupingChangeSpec(spec) && relatedDatabaseGroup"
      class="flex items-center gap-2 truncate"
    >
      <NTooltip>
        <template #trigger><DatabaseGroupIcon class="w-4 h-auto" /></template>
        {{ $t("dynamic.resource.database-group") }}
      </NTooltip>
      <span class="truncate text-sm">{{
        relatedDatabaseGroup.databaseGroupName
      }}</span>
    </div>
    <!-- Fallback -->
    <div v-else class="flex items-center gap-2 text-sm">Unknown type</div>
  </div>
</template>

<script setup lang="ts">
import { isEqual } from "lodash-es";
import { NTooltip } from "naive-ui";
import { computed, onMounted } from "vue";
import DatabaseGroupIcon from "@/components/DatabaseGroupIcon.vue";
import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import { InstanceV1Name } from "@/components/v2";
import { useDBGroupStore } from "@/store";
import { DatabaseGroupView } from "@/types/proto/v1/database_group_service";
import {
  PlanCheckRun_Result_Status,
  type Plan_Spec,
} from "@/types/proto/v1/plan_service";
import {
  databaseForSpec,
  isDatabaseChangeSpec,
  usePlanContext,
  isGroupingChangeSpec,
  planCheckRunListForSpec,
} from "../../logic";

const props = defineProps<{
  spec: Plan_Spec;
}>();

const { isCreating, plan, selectedSpec, events } = usePlanContext();
const dbGroupStore = useDBGroupStore();

const specClass = computed(() => {
  const classes: string[] = [];
  const isSelected = isEqual(props.spec, selectedSpec.value);
  if (isSelected) {
    classes.push("border-accent bg-accent bg-opacity-5 shadow");
  }
  if (planCheckStatus.value === PlanCheckRun_Result_Status.WARNING) {
    classes.push("bg-warning bg-opacity-5");
    if (isSelected) {
      classes.push("border-warning");
    }
  } else if (planCheckStatus.value === PlanCheckRun_Result_Status.ERROR) {
    classes.push("bg-error bg-opacity-5");
    if (isSelected) {
      classes.push("border-error");
    }
  }
  return classes;
});

const relatedDatabaseGroup = computed(() => {
  if (!isGroupingChangeSpec(props.spec)) {
    return undefined;
  }
  return dbGroupStore.getDBGroupByName(props.spec.changeDatabaseConfig!.target);
});

const planCheckStatus = computed((): PlanCheckRun_Result_Status => {
  if (isCreating.value) return PlanCheckRun_Result_Status.STATUS_UNSPECIFIED;

  const summary = planCheckRunSummaryForCheckRunList(
    planCheckRunListForSpec(plan.value, props.spec)
  );
  if (summary.errorCount > 0) {
    return PlanCheckRun_Result_Status.ERROR;
  }
  if (summary.warnCount > 0) {
    return PlanCheckRun_Result_Status.WARNING;
  }
  return PlanCheckRun_Result_Status.SUCCESS;
});

onMounted(async () => {
  if (isGroupingChangeSpec(props.spec)) {
    await dbGroupStore.getOrFetchDBGroupByName(
      props.spec.changeDatabaseConfig!.target,
      {
        skipCache: true,
        view: DatabaseGroupView.DATABASE_GROUP_VIEW_FULL,
      }
    );
  }
});

const onClickSpec = (spec: Plan_Spec) => {
  events.emit("select-spec", { spec });
};
</script>
