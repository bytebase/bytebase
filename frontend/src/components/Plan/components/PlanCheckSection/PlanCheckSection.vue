<template>
  <div v-if="shouldShow" class="pt-3 flex flex-col gap-y-1 overflow-hidden">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <h3 class="text-base">{{ $t("plan.checks.self") }}</h3>

        <NTooltip v-if="checksOfSelectedSpec.length > 0 && affectedRows > 0">
          <template #trigger>
            <NTag round :bordered="false">
              <div class="flex items-center gap-1">
                <span class="text-sm text-control-light">{{
                  $t("task.check-type.affected-rows.self")
                }}</span>
                <span class="text-sm">
                  {{ affectedRows }}
                </span>
                <CircleQuestionMarkIcon
                  class="size-[14px] text-control-light opacity-80"
                />
              </div>
            </NTag>
          </template>
          {{ $t("task.check-type.affected-rows.description") }}
        </NTooltip>
      </div>

      <div class="flex items-center gap-2">
        <!-- Status Summary -->
        <PlanCheckStatusCount
          :plan="plan"
          clickable
          @click="openChecksDrawer($event)"
        />

        <!-- Run Checks Button -->
        <NButton
          v-if="allowRunChecks"
          size="small"
          :loading="isRunningChecks"
          @click="runChecks"
        >
          <template #icon>
            <PlayIcon class="w-4 h-4" />
          </template>
          {{ $t("task.run-checks") }}
        </NButton>
      </div>
    </div>

    <ChecksDrawer
      v-model:show="showChecksDrawer"
      :status="selectedResultStatus"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import type { ConnectError } from "@connectrpc/connect";
import { PlayIcon, CircleQuestionMarkIcon } from "lucide-vue-next";
import { NButton, NTooltip, NTag } from "naive-ui";
import { computed, ref, watch } from "vue";
import { planServiceClientConnect } from "@/grpcweb";
import {
  useCurrentUserV1,
  useCurrentProjectV1,
  pushNotification,
  extractUserId,
} from "@/store";
import {
  PlanCheckRun_Result_Status,
  RunPlanChecksRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import {
  planCheckRunListForSpec,
  planSpecHasPlanChecks,
  usePlanContext,
  usePlanCheckStatus,
} from "../../logic";
import { useResourcePoller } from "../../logic/poller";
import ChecksDrawer from "../ChecksView/ChecksDrawer.vue";
import PlanCheckStatusCount from "../PlanCheckStatusCount.vue";
import { useSelectedSpec } from "../SpecDetailView/context";

const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const { plan, planCheckRuns } = usePlanContext();
const selectedSpec = useSelectedSpec();
const { refreshResources } = useResourcePoller();
const { statusCountString } = usePlanCheckStatus(plan);

const isRunningChecks = ref(false);
const showChecksDrawer = ref(false);
const selectedResultStatus = ref<PlanCheckRun_Result_Status>(
  PlanCheckRun_Result_Status.SUCCESS
);

const shouldShow = computed(() => {
  return planSpecHasPlanChecks(selectedSpec.value);
});

const allowRunChecks = computed(() => {
  const me = currentUser.value;
  if (extractUserId(plan.value.creator) === me.email) {
    return true;
  }
  return hasProjectPermissionV2(project.value, "bb.planCheckRuns.run");
});

const checksOfSelectedSpec = computed(() => {
  return planCheckRunListForSpec(planCheckRuns.value, selectedSpec.value);
});

const affectedRows = computed(() => {
  const summaryReportResults = checksOfSelectedSpec.value.filter((check) =>
    check.results.some((result) => result.report.case === "sqlSummaryReport")
  );
  return summaryReportResults.reduce((acc, check) => {
    if (check.results) {
      check.results.forEach((result) => {
        if (
          result.report?.case === "sqlSummaryReport" &&
          result.report.value.affectedRows !== undefined
        ) {
          acc += result.report.value.affectedRows;
        }
      });
    }
    return acc;
  }, 0n);
});

const runChecks = async () => {
  if (!plan.value.name || !selectedSpec.value) return;

  isRunningChecks.value = true;
  try {
    const request = create(RunPlanChecksRequestSchema, {
      name: plan.value.name,
    });
    await planServiceClientConnect.runPlanChecks(request);

    // After running checks, we need to refresh the plan and plan check runs.
    refreshResources(["plan", "planCheckRuns"], true /** force */);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: "Plan checks started",
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Failed to run plan checks",
      description: (error as ConnectError).message,
    });
  } finally {
    isRunningChecks.value = false;
  }
};

const openChecksDrawer = (status: PlanCheckRun_Result_Status) => {
  selectedResultStatus.value = status;
  showChecksDrawer.value = true;
};

// Prepare plan check runs.
watch(
  [() => selectedSpec.value.id, statusCountString],
  async () => {
    if (planSpecHasPlanChecks(selectedSpec.value)) {
      await refreshResources(["planCheckRuns"], true /** force */);
    }
  },
  {
    immediate: true,
  }
);
</script>
