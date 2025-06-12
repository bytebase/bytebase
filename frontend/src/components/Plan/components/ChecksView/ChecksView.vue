<template>
  <div class="h-full flex flex-col">
    <!-- Header with filters -->
    <div class="flex items-center justify-between px-4 py-2">
      <div class="flex items-center gap-4">
        <!-- Check result summary with icons -->
        <div class="flex items-center gap-3">
          <div
            v-if="statusCounts.success > 0"
            class="flex items-center gap-1 px-2 py-1 cursor-pointer"
            :class="[
              selectedStatus &&
                selectedStatus === PlanCheckRun_Result_Status.SUCCESS &&
                'bg-gray-100 rounded-lg',
              'text-lg text-success',
            ]"
            @click="toggleSelectedStatus(PlanCheckRun_Result_Status.SUCCESS)"
          >
            <CheckCircleIcon class="w-6 h-6" />
            <span>
              {{ $t("common.success") }}
            </span>
            <span class="font-semibold">
              {{ statusCounts.success }}
            </span>
          </div>
          <div
            v-if="statusCounts.warning > 0"
            class="flex items-center gap-1 px-2 py-1 cursor-pointer"
            :class="[
              selectedStatus &&
                selectedStatus === PlanCheckRun_Result_Status.WARNING &&
                'bg-gray-100 rounded-lg',
              'text-lg text-warning',
            ]"
            @click="toggleSelectedStatus(PlanCheckRun_Result_Status.WARNING)"
          >
            <AlertCircleIcon class="w-6 h-6" />
            <span>
              {{ $t("common.warning") }}
            </span>
            <span class="font-semibold">
              {{ statusCounts.warning }}
            </span>
          </div>
          <div
            v-if="statusCounts.error > 0"
            class="flex items-center gap-1 px-2 py-1 cursor-pointer"
            :class="[
              selectedStatus &&
                selectedStatus === PlanCheckRun_Result_Status.ERROR &&
                'bg-gray-100 rounded-lg',
              'text-lg text-error',
            ]"
            @click="toggleSelectedStatus(PlanCheckRun_Result_Status.ERROR)"
          >
            <XCircleIcon class="w-6 h-6" />
            <span>
              {{ $t("common.error") }}
            </span>
            <span class="font-semibold">
              {{ statusCounts.error }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Results List -->
    <div class="flex-1 overflow-y-auto border-t mt-2">
      <div v-if="isLoading" class="flex items-center justify-center py-12">
        <BBSpin />
      </div>

      <div
        v-else-if="filteredResults.length === 0"
        class="flex flex-col items-center justify-center py-12"
      >
        <CheckCircleIcon class="w-12 h-12 text-control-light opacity-50 mb-4" />
        <div class="text-lg font-medium text-control-light mb-2">
          {{
            hasFilters ? "No results match your filters" : "No check results"
          }}
        </div>
        <div class="text-sm text-control-lighter">
          {{
            hasFilters
              ? "Try adjusting your filters"
              : "Run checks to see results here"
          }}
        </div>
      </div>

      <div v-else class="divide-y">
        <!-- Group by check run -->
        <div
          v-for="checkRun in filteredCheckRuns"
          :key="checkRun.name"
          class="px-6 py-4"
        >
          <!-- Check Run Header -->
          <div class="flex items-start justify-between mb-2">
            <div class="flex items-center gap-3">
              <component
                :is="getCheckTypeIcon(checkRun.type)"
                class="w-5 h-5 text-control-light"
              />
              <div>
                <span class="text-sm font-medium">
                  {{ getCheckTypeLabel(checkRun.type) }}
                </span>
                <span class="text-xs text-control-light">
                  Target: {{ formatTarget(checkRun.target) }}
                </span>
              </div>
            </div>

            <div class="flex items-center gap-2">
              <NBadge
                :type="getStatusBadgeType(getCheckRunStatus(checkRun))"
                :value="getStatusLabel(getCheckRunStatus(checkRun))"
              />
              <span class="text-xs text-control-lighter">
                {{ formatTime(checkRun.createTime) }}
              </span>
            </div>
          </div>

          <!-- Results for this check run -->
          <div class="space-y-2 pl-8">
            <div
              v-for="(result, idx) in getFilteredResults(checkRun)"
              :key="idx"
              class="flex items-start gap-3 px-3 py-2 border rounded-lg bg-gray-50"
            >
              <component
                :is="getStatusIcon(result.status)"
                class="w-4 h-4 mt-0.5 flex-shrink-0"
                :class="getStatusColor(result.status)"
              />

              <div class="flex-1 min-w-0">
                <div class="text-sm font-medium text-main">
                  {{ result.title }}
                </div>
                <div
                  v-if="result.content"
                  class="text-xs text-control-light mt-1"
                >
                  {{ result.content }}
                </div>
                <div
                  v-if="
                    result.sqlReviewReport && result.sqlReviewReport.line > 0
                  "
                  class="text-xs text-control-lighter mt-1"
                >
                  Line {{ result.sqlReviewReport.line }}, Column
                  {{ result.sqlReviewReport.column }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  CheckCircleIcon,
  AlertCircleIcon,
  XCircleIcon,
  FileCodeIcon,
  DatabaseIcon,
  ShieldIcon,
  SearchCodeIcon,
} from "lucide-vue-next";
import { NBadge } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import {
  PlanCheckRun_Result_Status,
  PlanCheckRun_Type,
  type PlanCheckRun,
} from "@/types/proto/v1/plan_service";
import { extractDatabaseResourceName, humanizeTs } from "@/utils";
import { usePlanContext } from "../../logic/context";

const { t } = useI18n();
const { planCheckRunList } = usePlanContext();

const isLoading = ref(false);
const selectedStatus = ref<PlanCheckRun_Result_Status | undefined>(undefined);

const hasFilters = computed(() => {
  return selectedStatus.value !== undefined;
});

const filteredCheckRuns = computed(() => {
  return planCheckRunList.value.filter((checkRun) => {
    // Filter by status - check if any result matches the selected status
    if (selectedStatus.value !== undefined) {
      const hasMatchingResult = checkRun.results.some(
        (result) => result.status === selectedStatus.value
      );
      if (!hasMatchingResult) {
        return false;
      }
    }

    return true;
  });
});

const filteredResults = computed(() => {
  const results: Array<{ checkRun: PlanCheckRun; result: any }> = [];

  for (const checkRun of filteredCheckRuns.value) {
    for (const result of checkRun.results) {
      if (
        selectedStatus.value === undefined ||
        result.status === selectedStatus.value
      ) {
        results.push({ checkRun, result });
      }
    }
  }

  return results;
});

const statusCounts = computed(() => {
  const counts = {
    success: 0,
    warning: 0,
    error: 0,
  };

  for (const checkRun of planCheckRunList.value) {
    for (const result of checkRun.results) {
      switch (result.status) {
        case PlanCheckRun_Result_Status.SUCCESS:
          counts.success++;
          break;
        case PlanCheckRun_Result_Status.WARNING:
          counts.warning++;
          break;
        case PlanCheckRun_Result_Status.ERROR:
          counts.error++;
          break;
      }
    }
  }

  return counts;
});

const toggleSelectedStatus = (status: PlanCheckRun_Result_Status) => {
  if (selectedStatus.value === status) {
    selectedStatus.value = undefined; // Deselect if already selected
  } else {
    selectedStatus.value = status; // Select the new status
  }
};

const getFilteredResults = (checkRun: PlanCheckRun) => {
  return checkRun.results.filter((result) => {
    return (
      selectedStatus.value === undefined ||
      result.status === selectedStatus.value
    );
  });
};

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

const getCheckTypeIcon = (type: PlanCheckRun_Type) => {
  switch (type) {
    case PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE:
      return SearchCodeIcon;
    case PlanCheckRun_Type.DATABASE_STATEMENT_SUMMARY_REPORT:
      return FileCodeIcon;
    case PlanCheckRun_Type.DATABASE_CONNECT:
      return DatabaseIcon;
    case PlanCheckRun_Type.DATABASE_GHOST_SYNC:
      return ShieldIcon;
    default:
      return FileCodeIcon;
  }
};

const getCheckTypeLabel = (type: PlanCheckRun_Type) => {
  switch (type) {
    case PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE:
      return t("task.check-type.sql-review");
    case PlanCheckRun_Type.DATABASE_STATEMENT_SUMMARY_REPORT:
      return t("task.check-type.summary-report");
    case PlanCheckRun_Type.DATABASE_CONNECT:
      return t("task.check-type.connection");
    case PlanCheckRun_Type.DATABASE_GHOST_SYNC:
      return t("task.check-type.ghost-sync");
    default:
      return type.toString();
  }
};

const getStatusIcon = (status: PlanCheckRun_Result_Status) => {
  switch (status) {
    case PlanCheckRun_Result_Status.ERROR:
      return XCircleIcon;
    case PlanCheckRun_Result_Status.WARNING:
      return AlertCircleIcon;
    case PlanCheckRun_Result_Status.SUCCESS:
      return CheckCircleIcon;
    default:
      return CheckCircleIcon;
  }
};

const getStatusColor = (status: PlanCheckRun_Result_Status) => {
  switch (status) {
    case PlanCheckRun_Result_Status.ERROR:
      return "text-error";
    case PlanCheckRun_Result_Status.WARNING:
      return "text-warning";
    case PlanCheckRun_Result_Status.SUCCESS:
      return "text-success";
    default:
      return "text-control";
  }
};

const getStatusBadgeType = (status: PlanCheckRun_Result_Status) => {
  switch (status) {
    case PlanCheckRun_Result_Status.ERROR:
      return "error";
    case PlanCheckRun_Result_Status.WARNING:
      return "warning";
    case PlanCheckRun_Result_Status.SUCCESS:
      return "success";
    default:
      return "default";
  }
};

const getStatusLabel = (status: PlanCheckRun_Result_Status) => {
  switch (status) {
    case PlanCheckRun_Result_Status.ERROR:
      return "Error";
    case PlanCheckRun_Result_Status.WARNING:
      return "Warning";
    case PlanCheckRun_Result_Status.SUCCESS:
      return "Success";
    default:
      return "Unknown";
  }
};

const formatTarget = (target: string): string => {
  const { instanceName, databaseName } = extractDatabaseResourceName(target);
  if (instanceName && databaseName) {
    return `${databaseName} (${instanceName})`;
  }
  return target;
};

const formatTime = (timestamp: any): string => {
  if (!timestamp) return "";
  return humanizeTs(new Date(timestamp.seconds * 1000).getTime() / 1000);
};
</script>
