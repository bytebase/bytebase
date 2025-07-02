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
            @click="openDrawer('ERROR')"
          >
            <XCircleIcon class="w-5 h-5 text-error" />
            <span class="text-error">{{ summary.error }}</span>
          </button>
          <button
            v-if="summary.warning > 0"
            class="flex items-center gap-1 hover:opacity-80"
            @click="openDrawer('WARNING')"
          >
            <AlertCircleIcon class="w-5 h-5 text-warning" />
            <span class="text-warning">{{ summary.warning }}</span>
          </button>
          <button
            v-if="summary.success > 0"
            class="flex items-center gap-1 hover:opacity-80"
            @click="openDrawer('SUCCESS')"
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

    <!-- Status Drawer -->
    <Drawer v-model:show="drawerVisible">
      <DrawerContent
        :title="drawerTitle"
        class="w-[40rem] max-w-[100vw] relative"
      >
        <div class="w-full h-full flex flex-col">
          <!-- Drawer Header -->
          <div class="flex items-center justify-between px-4 py-3 border-b">
            <div class="flex items-center gap-2">
              <component
                :is="getStatusIcon(selectedStatus)"
                class="w-5 h-5"
                :class="getStatusColor(selectedStatus)"
              />
              <h3 class="text-lg font-medium">{{ drawerTitle }}</h3>
            </div>
          </div>

          <!-- Drawer Content -->
          <div class="flex-1 overflow-y-auto px-4 py-3">
            <div v-if="drawerCheckRuns.length > 0" class="space-y-4">
              <!-- Group by Check Type -->
              <div
                v-for="[checkType, checkRuns] in drawerCheckRunsByType"
                :key="checkType"
                class="space-y-2"
              >
                <div class="flex items-center gap-2 mb-3">
                  <component
                    :is="getCheckTypeIcon(checkType)"
                    class="w-5 h-5"
                  />
                  <span class="font-medium">{{
                    getCheckTypeLabel(checkType)
                  }}</span>
                  <span class="text-control-light"
                    >({{ checkRuns.length }})</span
                  >
                </div>

                <div class="space-y-3 pl-6">
                  <div
                    v-for="checkRun in checkRuns"
                    :key="checkRun.name"
                    class="space-y-2"
                  >
                    <div class="text-sm font-medium text-control">
                      <DatabaseDisplay :database="checkRun.target" />
                    </div>
                    <div
                      v-for="(result, idx) in getResultsByStatus(
                        checkRun,
                        selectedStatus
                      )"
                      :key="idx"
                      class="px-3 py-1 border rounded-lg bg-gray-50"
                    >
                      <div
                        class="text-sm"
                        :class="getStatusColor(selectedStatus)"
                      >
                        {{ result.title }}
                      </div>
                      <div
                        v-if="result.content"
                        class="text-xs text-control-light mt-0.5"
                      >
                        {{ result.content }}
                      </div>
                      <div
                        v-if="
                          (result as any).sqlReviewReport &&
                          (result as any).sqlReviewReport?.line > 0
                        "
                        class="text-xs text-control-lighter mt-0.5"
                      >
                        Line {{ (result as any).sqlReviewReport?.line }}, Column
                        {{ (result as any).sqlReviewReport?.column }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div v-else class="text-center py-8 text-control-light">
              No {{ selectedStatus.toLowerCase() }} results
            </div>
          </div>
        </div>
      </DrawerContent>
    </Drawer>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import {
  CheckCircleIcon,
  AlertCircleIcon,
  XCircleIcon,
  PlayIcon,
  FileCodeIcon,
  DatabaseIcon,
  ShieldIcon,
  SearchCodeIcon,
} from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import Drawer from "@/components/v2/Container/Drawer.vue";
import DrawerContent from "@/components/v2/Container/DrawerContent.vue";
import { planServiceClientConnect } from "@/grpcweb";
import {
  useCurrentUserV1,
  useCurrentProjectV1,
  pushNotification,
  extractUserId,
} from "@/store";
import { RunPlanChecksRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import {
  PlanCheckRun_Result_Status,
  type PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { planCheckRunListForSpec, planSpecHasPlanChecks } from "../../logic";
import { usePlanContext } from "../../logic/context";
import { usePlanSpecContext } from "../SpecDetailView/context";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const { plan, planCheckRuns } = usePlanContext();
const { selectedSpec } = usePlanSpecContext();

const isRunningChecks = ref(false);
const drawerVisible = ref(false);
const selectedStatus = ref<"ERROR" | "WARNING" | "SUCCESS">("ERROR");

const show = computed(() => {
  return planSpecHasPlanChecks(selectedSpec.value);
});

const checkRunsForSpec = computed(() => {
  return planCheckRunListForSpec(planCheckRuns.value, selectedSpec.value);
});

const summary = computed(() => {
  const result = { success: 0, warning: 0, error: 0 };
  for (const checkRun of checkRunsForSpec.value) {
    const status = getCheckRunStatus(checkRun);
    if (status === PlanCheckRun_Result_Status.ERROR) {
      result.error++;
    } else if (status === PlanCheckRun_Result_Status.WARNING) {
      result.warning++;
    } else if (status === PlanCheckRun_Result_Status.SUCCESS) {
      result.success++;
    }
  }
  return result;
});

const drawerTitle = computed(() => {
  if (selectedStatus.value === "ERROR") return "Error Details";
  if (selectedStatus.value === "WARNING") return "Warning Details";
  return "Success Details";
});

const drawerCheckRuns = computed(() => {
  const result: PlanCheckRun[] = [];

  for (const checkRun of checkRunsForSpec.value) {
    const status = getCheckRunStatus(checkRun);
    if (
      (selectedStatus.value === "ERROR" &&
        status === PlanCheckRun_Result_Status.ERROR) ||
      (selectedStatus.value === "WARNING" &&
        status === PlanCheckRun_Result_Status.WARNING) ||
      (selectedStatus.value === "SUCCESS" &&
        status === PlanCheckRun_Result_Status.SUCCESS)
    ) {
      result.push(checkRun);
    }
  }

  return result;
});

const drawerCheckRunsByType = computed(() => {
  const groups = new Map<string, PlanCheckRun[]>();

  for (const checkRun of drawerCheckRuns.value) {
    const type = String(checkRun.type);
    if (!groups.has(type)) {
      groups.set(type, []);
    }
    groups.get(type)!.push(checkRun);
  }

  return groups;
});

const allowRunChecks = computed(() => {
  const me = currentUser.value;
  if (extractUserId(plan.value.creator) === me.email) {
    return true;
  }

  return hasProjectPermissionV2(project.value, "bb.planCheckRuns.run");
});

const getCheckRunStatus = (
  checkRun: PlanCheckRun
): PlanCheckRun_Result_Status => {
  let hasError = false;
  let hasWarning = false;

  for (const result of checkRun.results) {
    if (result.status === PlanCheckRun_Result_Status.ERROR) {
      hasError = true;
    } else if (result.status === PlanCheckRun_Result_Status.WARNING) {
      hasWarning = true;
    }
  }

  if (hasError) return PlanCheckRun_Result_Status.ERROR;
  if (hasWarning) return PlanCheckRun_Result_Status.WARNING;
  return PlanCheckRun_Result_Status.SUCCESS;
};

const getCheckTypeIcon = (type: string) => {
  switch (type) {
    case "DATABASE_STATEMENT_ADVISE":
      return SearchCodeIcon;
    case "DATABASE_STATEMENT_SUMMARY_REPORT":
      return FileCodeIcon;
    case "DATABASE_CONNECT":
      return DatabaseIcon;
    case "DATABASE_GHOST_SYNC":
      return ShieldIcon;
    default:
      return FileCodeIcon;
  }
};

const getCheckTypeLabel = (type: string) => {
  switch (type) {
    case "DATABASE_STATEMENT_ADVISE":
      return t("task.check-type.sql-review");
    case "DATABASE_STATEMENT_SUMMARY_REPORT":
      return t("task.check-type.summary-report");
    case "DATABASE_CONNECT":
      return t("task.check-type.connection");
    case "DATABASE_GHOST_SYNC":
      return t("task.check-type.ghost-sync");
    default:
      return type;
  }
};

const runChecks = async () => {
  if (!plan.value.name || !selectedSpec.value) return;

  isRunningChecks.value = true;
  try {
    const request = create(RunPlanChecksRequestSchema, {
      name: plan.value.name,
    });
    await planServiceClientConnect.runPlanChecks(request);
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

const openDrawer = (status: "ERROR" | "WARNING" | "SUCCESS") => {
  selectedStatus.value = status;
  drawerVisible.value = true;
};

const getStatusIcon = (status: string) => {
  if (status === "ERROR") return XCircleIcon;
  if (status === "WARNING") return AlertCircleIcon;
  return CheckCircleIcon;
};

const getStatusColor = (status: string) => {
  if (status === "ERROR") return "text-error";
  if (status === "WARNING") return "text-warning";
  return "text-success";
};

const getResultsByStatus = (checkRun: PlanCheckRun, status: string) => {
  return checkRun.results.filter(
    (result) =>
      result.status ===
      PlanCheckRun_Result_Status[
        status as keyof typeof PlanCheckRun_Result_Status
      ]
  );
};
</script>
