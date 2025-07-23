<template>
  <div
    v-if="shouldShow"
    class="px-4 pt-3 flex flex-col gap-y-1 overflow-hidden"
  >
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-1">
        <h3 class="text-base font-medium">{{ $t("plan.checks.self") }}</h3>
      </div>

      <div class="flex items-center gap-2">
        <!-- Status Summary -->
        <div class="flex items-center gap-2 text-sm">
          <button
            v-if="getChecksCount(PlanCheckRun_Result_Status.ERROR) > 0"
            class="flex items-center gap-1 hover:opacity-80"
            @click="openChecksDrawer(PlanCheckRun_Result_Status.ERROR)"
          >
            <XCircleIcon class="w-5 h-5 text-error" />
            <span class="text-error">{{
              getChecksCount(PlanCheckRun_Result_Status.ERROR)
            }}</span>
          </button>
          <button
            v-if="getChecksCount(PlanCheckRun_Result_Status.WARNING) > 0"
            class="flex items-center gap-1 hover:opacity-80"
            @click="openChecksDrawer(PlanCheckRun_Result_Status.WARNING)"
          >
            <AlertCircleIcon class="w-5 h-5 text-warning" />
            <span class="text-warning">{{
              getChecksCount(PlanCheckRun_Result_Status.WARNING)
            }}</span>
          </button>
          <button
            v-if="getChecksCount(PlanCheckRun_Result_Status.SUCCESS) > 0"
            class="flex items-center gap-1 hover:opacity-80"
            @click="openChecksDrawer(PlanCheckRun_Result_Status.SUCCESS)"
          >
            <CheckCircleIcon class="w-5 h-5 text-success" />
            <span class="text-success">{{
              getChecksCount(PlanCheckRun_Result_Status.SUCCESS)
            }}</span>
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

    <ChecksDrawer
      v-model:show="showChecksDrawer"
      :status="selectedResultStatus"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import type { ConnectError } from "@connectrpc/connect";
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
import { useSelectedSpec } from "../SpecDetailView/context";

const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const { plan } = usePlanContext();
const selectedSpec = useSelectedSpec();
const { refreshResources } = useResourcePoller();

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

const getChecksCount = (status: PlanCheckRun_Result_Status) => {
  return (
    plan.value.planCheckRunStatusCount[PlanCheckRun_Result_Status[status]] || 0
  );
};

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
</script>
