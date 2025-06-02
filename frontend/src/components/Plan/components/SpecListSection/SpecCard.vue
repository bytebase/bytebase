<template>
  <div
    v-for="target in targetsForSpec(spec)"
    :key="`${spec.id}-${target}`"
    class="px-3 py-2 w-full cursor-pointer border rounded lg:flex-1 flex justify-start items-center overflow-hidden gap-x-1"
    :class="[getTargetClass(target)]"
    @click="onClickSpec(spec, target)"
  >
    <NTooltip
      v-if="
        [
          PlanCheckRun_Result_Status.WARNING,
          PlanCheckRun_Result_Status.ERROR,
        ].includes(planCheckStatusForTarget(target))
      "
      trigger="hover"
      placement="top"
    >
      <template #trigger>
        <AlertCircleIcon
          class="w-4 h-4 shrink-0"
          :class="[
            planCheckStatusForTarget(target) ===
            PlanCheckRun_Result_Status.ERROR
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
      class="flex items-center gap-1 truncate"
    >
      <InstanceV1Name
        :instance="getDatabaseForTarget(target).instanceResource"
        :link="false"
        class="text-gray-500 text-sm"
      />
      <span class="truncate text-sm">{{
        getDatabaseForTarget(target).databaseName
      }}</span>
    </div>
    <div
      v-else-if="isGroupingChangeSpec(spec) && getRelatedDatabaseGroup(target)"
      class="flex items-center gap-2 truncate"
    >
      <NTooltip>
        <template #trigger>
          <DatabaseGroupIcon class="w-4 h-auto" />
        </template>
        {{ $t("dynamic.resource.database-group") }}
      </NTooltip>
      <span class="truncate text-sm">{{
        getRelatedDatabaseGroup(target)?.title
      }}</span>
    </div>
    <!-- Fallback -->
    <div v-else class="flex items-center gap-2 text-sm">Unknown target</div>
  </div>
</template>

<script setup lang="ts">
import { isEqual } from "lodash-es";
import { AlertCircleIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import DatabaseGroupIcon from "@/components/DatabaseGroupIcon.vue";
import { mockDatabase } from "@/components/IssueV1/logic/utils";
import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import { InstanceV1Name } from "@/components/v2";
import {
  useCurrentProjectV1,
  useDBGroupStore,
  useDatabaseV1Store,
} from "@/store";
import { isValidDatabaseName } from "@/types";
import {
  PlanCheckRun_Result_Status,
  type Plan_Spec,
} from "@/types/proto/v1/plan_service";
import { sheetNameOfSpec } from "@/utils";
import {
  isDatabaseChangeSpec,
  usePlanContext,
  isGroupingChangeSpec,
  targetsForSpec,
} from "../../logic";

const props = defineProps<{
  spec: Plan_Spec;
}>();

const { project } = useCurrentProjectV1();
const { isCreating, plan, selectedSpec, selectedTarget, events } =
  usePlanContext();
const dbGroupStore = useDBGroupStore();

const getTargetClass = (target: string) => {
  const classes: string[] = [];
  const isSpecSelected = isEqual(props.spec, selectedSpec.value);
  const isTargetSelected = isSpecSelected && selectedTarget.value === target;

  if (isTargetSelected) {
    classes.push("border-accent bg-accent bg-opacity-5 shadow");
  }

  const checkStatus = planCheckStatusForTarget(target);
  if (checkStatus === PlanCheckRun_Result_Status.WARNING) {
    classes.push("bg-warning bg-opacity-5");
    if (isTargetSelected) {
      classes.push("border-warning");
    }
  } else if (checkStatus === PlanCheckRun_Result_Status.ERROR) {
    classes.push("bg-error bg-opacity-5");
    if (isTargetSelected) {
      classes.push("border-error");
    }
  }

  return classes;
};

const getDatabaseForTarget = (target: string) => {
  const db = useDatabaseV1Store().getDatabaseByName(target);
  if (!isValidDatabaseName(db.name)) {
    return mockDatabase(project.value, target);
  }
  return db;
};

const getRelatedDatabaseGroup = (target: string) => {
  if (!isGroupingChangeSpec(props.spec)) {
    return undefined;
  }
  return dbGroupStore.getDBGroupByName(target);
};

const planCheckStatusForTarget = (
  target: string
): PlanCheckRun_Result_Status => {
  if (isCreating.value) return PlanCheckRun_Result_Status.STATUS_UNSPECIFIED;
  const planCheckRuns = plan.value.planCheckRunList.filter(
    (run) => run.sheet === sheetNameOfSpec(props.spec) && run.target === target
  );
  const summary = planCheckRunSummaryForCheckRunList(planCheckRuns);
  if (summary.errorCount > 0) {
    return PlanCheckRun_Result_Status.ERROR;
  }
  if (summary.warnCount > 0) {
    return PlanCheckRun_Result_Status.WARNING;
  }
  return PlanCheckRun_Result_Status.SUCCESS;
};

const onClickSpec = (spec: Plan_Spec, target: string) => {
  events.emit("select-spec", { spec });
  events.emit("select-target", { target });
};
</script>
