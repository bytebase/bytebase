<template>
  <div v-if="shouldShow" class="py-3 flex flex-col gap-y-2 overflow-hidden">
    <!-- Row 1: Title + Run button -->
    <div class="flex items-center justify-between gap-2">
      <div class="flex flex-row items-center gap-2">
        <h3 class="text-base">{{ $t("plan.checks.self") }}</h3>
        <NTooltip v-if="isStatementOversized">
          <template #trigger>
            <NTag type="warning" round size="tiny">
              <template #icon>
                <CircleQuestionMarkIcon class="size-3" />
              </template>
              {{ $t("common.skipped") }}
            </NTag>
          </template>
          {{ $t("issue.sql-check.statement-is-too-large") }}
        </NTooltip>
      </div>
      <NButton
        v-if="allowRunChecks"
        size="tiny"
        :loading="isRunningChecks"
        @click="runChecks"
      >
        <template #icon>
          <PlayIcon class="w-4 h-4" />
        </template>
        {{ $t("common.run") }}
      </NButton>
    </div>

    <!-- Row 2: Status badges -->
    <div class="flex items-center flex-wrap gap-2 min-w-0">
      <PlanCheckStatusCount
        :plan="plan"
        :show-label="true"
        clickable
        @click="openChecksDrawer($event)"
      />

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
                class="size-3.5 text-control-light opacity-80"
              />
            </div>
          </NTag>
        </template>
        {{ $t("task.check-type.affected-rows.description") }}
      </NTooltip>
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
import { CircleQuestionMarkIcon, PlayIcon } from "lucide-vue-next";
import { NButton, NTag, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { STATEMENT_SKIP_CHECK_THRESHOLD } from "@/components/SQLCheck/common";
import { planServiceClientConnect } from "@/connect";
import {
  extractUserId,
  pushNotification,
  useCurrentProjectV1,
  useCurrentUserV1,
} from "@/store";
import { RunPlanChecksRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import {
  planCheckRunListForSpec,
  planSpecHasPlanChecks,
  usePlanCheckStatus,
  usePlanContext,
} from "../../logic";
import { useResourcePoller } from "../../logic/poller";
import ChecksDrawer from "../ChecksView/ChecksDrawer.vue";
import PlanCheckStatusCount from "../PlanCheckStatusCount.vue";
import { useSelectedSpec } from "../SpecDetailView/context";
import { useSpecSheet } from "../StatementSection/useSpecSheet";

const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const { plan, planCheckRuns } = usePlanContext();
const { selectedSpec } = useSelectedSpec();
const { refreshResources } = useResourcePoller();
const { statusCountString } = usePlanCheckStatus(plan);
const { sheet } = useSpecSheet(selectedSpec);

const isRunningChecks = ref(false);
const showChecksDrawer = ref(false);
const selectedResultStatus = ref<Advice_Level>(Advice_Level.SUCCESS);

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

const isStatementOversized = computed(() => {
  if (!sheet.value) return false;
  return sheet.value.contentSize > STATEMENT_SKIP_CHECK_THRESHOLD;
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

const openChecksDrawer = (status: Advice_Level) => {
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
