<template>
  <div v-if="show" class="px-4 pt-3 flex flex-col gap-y-1 overflow-hidden">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-1">
        <h3 class="text-base font-medium">{{ $t("plan.checks.self") }}</h3>
      </div>

      <div class="flex items-center gap-2">
        <!-- Status Summary -->
        <div class="flex items-center gap-2 text-sm">
          <button
            v-if="summary.error > 0"
            class="flex items-center gap-1 hover:opacity-80"
            @click="openDrawer(PlanCheckRun_Result_Status.ERROR)"
          >
            <XCircleIcon class="w-5 h-5 text-error" />
            <span class="text-error">{{ summary.error }}</span>
          </button>
          <button
            v-if="summary.warning > 0"
            class="flex items-center gap-1 hover:opacity-80"
            @click="openDrawer(PlanCheckRun_Result_Status.WARNING)"
          >
            <AlertCircleIcon class="w-5 h-5 text-warning" />
            <span class="text-warning">{{ summary.warning }}</span>
          </button>
          <button
            v-if="summary.success > 0"
            class="flex items-center gap-1 hover:opacity-80"
            @click="openDrawer(PlanCheckRun_Result_Status.SUCCESS)"
          >
            <CheckCircleIcon class="w-5 h-5 text-success" />
            <span class="text-success">{{ summary.success }}</span>
          </button>
        </div>

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

    <ChecksDrawer v-model:show="drawerVisible" :status="selectedStatus" />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { isNumber } from "lodash-es";
import {
  CheckCircleIcon,
  AlertCircleIcon,
  XCircleIcon,
  PlayIcon,
} from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
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
import { planSpecHasPlanChecks } from "../../logic";
import { usePlanContext } from "../../logic/context";
import { useResourcePoller } from "../../logic/poller";
import ChecksDrawer from "../ChecksView/ChecksDrawer.vue";
import { usePlanSpecContext } from "../SpecDetailView/context";

const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const { plan } = usePlanContext();
const { selectedSpec } = usePlanSpecContext();
const { requestEnhancedPolling } = useResourcePoller();

const isRunningChecks = ref(false);
const drawerVisible = ref(false);
const selectedStatus = ref<PlanCheckRun_Result_Status>(
  PlanCheckRun_Result_Status.SUCCESS
);

const show = computed(() => {
  return planSpecHasPlanChecks(selectedSpec.value);
});

const summary = computed(() => {
  const result = { success: 0, warning: 0, error: 0 };
  if (
    plan.value.planCheckRunStatusCount["ERROR"] &&
    isNumber(plan.value.planCheckRunStatusCount["ERROR"])
  ) {
    result.error = plan.value.planCheckRunStatusCount["ERROR"];
  }
  if (
    plan.value.planCheckRunStatusCount["WARNING"] &&
    isNumber(plan.value.planCheckRunStatusCount["WARNING"])
  ) {
    result.warning = plan.value.planCheckRunStatusCount["WARNING"];
  }
  if (
    plan.value.planCheckRunStatusCount["SUCCESS"] &&
    isNumber(plan.value.planCheckRunStatusCount["SUCCESS"])
  ) {
    result.success = plan.value.planCheckRunStatusCount["SUCCESS"];
  }
  return result;
});

const allowRunChecks = computed(() => {
  const me = currentUser.value;
  if (extractUserId(plan.value.creator) === me.email) {
    return true;
  }

  return hasProjectPermissionV2(project.value, "bb.planCheckRuns.run");
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
    requestEnhancedPolling(["plan", "planCheckRuns"], true /** once */);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: "Plan checks started",
    });
  } catch (error: any) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Failed to run plan checks",
      description: error.message,
    });
  } finally {
    isRunningChecks.value = false;
  }
};

const openDrawer = (status: PlanCheckRun_Result_Status) => {
  // Fetch the latest plan check runs for the selected status.
  requestEnhancedPolling(["planCheckRuns"], true /** once */);
  selectedStatus.value = status;
  drawerVisible.value = true;
};
</script>
