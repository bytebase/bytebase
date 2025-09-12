<template>
  <div class="flex-1 flex flex-col">
    <!-- Header with status counts -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <div class="flex items-center gap-3" v-if="hasAnyStatus">
          <div
            v-if="statusSummary.error > 0"
            class="flex items-center gap-1 text-error cursor-pointer"
            :class="getItemClass(PlanCheckRun_Result_Status.ERROR)"
            @click="toggleSelectedStatus(PlanCheckRun_Result_Status.ERROR)"
          >
            <XCircleIcon class="w-5 h-5" />
            <span>{{ $t("common.error") }}</span>
            <span>{{ statusSummary.error }}</span>
          </div>
          <div
            v-if="statusSummary.warning > 0"
            class="flex items-center gap-1 text-warning cursor-pointer"
            :class="getItemClass(PlanCheckRun_Result_Status.WARNING)"
            @click="toggleSelectedStatus(PlanCheckRun_Result_Status.WARNING)"
          >
            <AlertCircleIcon class="w-5 h-5" />
            <span>{{ $t("common.warning") }}</span>
            <span>{{ statusSummary.warning }}</span>
          </div>
          <div
            v-if="statusSummary.success > 0"
            class="flex items-center gap-1 text-success cursor-pointer"
            :class="getItemClass(PlanCheckRun_Result_Status.SUCCESS)"
            @click="toggleSelectedStatus(PlanCheckRun_Result_Status.SUCCESS)"
          >
            <CheckCircleIcon class="w-5 h-5" />
            <span>{{ $t("common.success") }}</span>
            <span>{{ statusSummary.success }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Results List -->
    <div class="flex-1 overflow-y-auto">
      <div
        v-if="isLoading"
        class="flex flex-col items-center justify-center py-12"
      >
        <BBSpin />
      </div>
      <!-- Empty state -->
      <div
        v-else-if="filteredCheckRuns.length === 0"
        class="flex flex-col items-center justify-center py-12"
      >
        <CheckCircleIcon class="w-12 h-12 text-control-light opacity-50 mb-4" />
        <div class="text-lg text-control-light">
          {{
            hasFilters ? "No results match your filters" : "No check results"
          }}
        </div>
      </div>
      <div v-else class="divide-y">
        <!-- Group by check run -->
        <div
          v-for="checkRun in filteredCheckRuns"
          :key="checkRun.name"
          class="px-2 py-4"
        >
          <!-- Check Run Header -->
          <div class="flex items-start justify-between mb-2">
            <div class="flex items-center gap-3">
              <component
                :is="getCheckTypeIcon(checkRun.type)"
                class="w-5 h-5 text-control-light"
              />
              <div class="flex flex-row items-center gap-2">
                <span class="text-sm font-medium">
                  {{ getCheckTypeLabel(checkRun.type) }}
                </span>
                <NTooltip v-if="getCheckTypeDescription(checkRun.type)">
                  <template #trigger>
                    <CircleQuestionMarkIcon
                      class="w-4 h-4 text-control-light"
                    />
                  </template>
                  {{ getCheckTypeDescription(checkRun.type) }}
                </NTooltip>
                <span class="text-sm text-control-light">
                  {{ formatTime(checkRun.createTime) }}
                </span>
              </div>
            </div>

            <div class="flex items-center gap-2">
              <DatabaseDisplay :database="checkRun.target" />
            </div>
          </div>

          <!-- Results for this check run -->
          <div class="space-y-2 pl-8">
            <CheckResultItem
              v-for="(result, idx) in getFilteredResults(checkRun)"
              :key="idx"
              :status="getCheckResultStatus(result.status)"
              :title="result.title"
              :content="result.content"
              :code="result.code"
              :report-type="result.report?.case"
              :position="
                result.report?.case === 'sqlReviewReport' &&
                result.report.value.startPosition
                  ? result.report.value.startPosition
                  : undefined
              "
              :affected-rows="
                result.report?.case === 'sqlSummaryReport'
                  ? result.report.value.affectedRows
                  : undefined
              "
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import {
  CheckCircleIcon,
  FileCodeIcon,
  DatabaseIcon,
  ShieldIcon,
  SearchCodeIcon,
  CircleQuestionMarkIcon,
  XCircleIcon,
  AlertCircleIcon,
} from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import {
  PlanCheckRun_Result_Status,
  PlanCheckRun_Type,
  type PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import { humanizeTs } from "@/utils";
import CheckResultItem from "../common/CheckResultItem.vue";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";

const props = defineProps<{
  defaultStatus?: PlanCheckRun_Result_Status;
  planCheckRuns: PlanCheckRun[];
  isLoading?: boolean;
}>();

const { t } = useI18n();

const selectedStatus = ref<PlanCheckRun_Result_Status | undefined>(
  props.defaultStatus
);

const hasFilters = computed(() => {
  return selectedStatus.value !== undefined;
});

// Calculate status summary from plan check runs
const statusSummary = computed(() => {
  const summary = { error: 0, warning: 0, success: 0 };

  for (const checkRun of props.planCheckRuns) {
    for (const result of checkRun.results) {
      if (result.status === PlanCheckRun_Result_Status.ERROR) {
        summary.error++;
      } else if (result.status === PlanCheckRun_Result_Status.WARNING) {
        summary.warning++;
      } else if (result.status === PlanCheckRun_Result_Status.SUCCESS) {
        summary.success++;
      }
    }
  }

  return summary;
});

const hasAnyStatus = computed(() => {
  return (
    statusSummary.value.error > 0 ||
    statusSummary.value.warning > 0 ||
    statusSummary.value.success > 0
  );
});

const toggleSelectedStatus = (status: PlanCheckRun_Result_Status) => {
  if (selectedStatus.value === status) {
    selectedStatus.value = undefined; // Deselect if already selected
  } else {
    selectedStatus.value = status; // Select the new status
  }
};

const getItemClass = (status: PlanCheckRun_Result_Status) => {
  const classes: string[] = [];

  if (selectedStatus.value === status) {
    classes.push("bg-gray-100 rounded-lg px-2 py-1");
  } else {
    // Add some padding to align with selected items
    classes.push("px-2 py-1");
  }

  return classes;
};

const filteredCheckRuns = computed(() => {
  return props.planCheckRuns.filter((checkRun) => {
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

const getFilteredResults = (checkRun: PlanCheckRun) => {
  return checkRun.results.filter((result) => {
    return (
      selectedStatus.value === undefined ||
      result.status === selectedStatus.value
    );
  });
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
      return t("task.check-type.sql-review.self");
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

const getCheckTypeDescription = (type: PlanCheckRun_Type) => {
  switch (type) {
    case PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE:
      return t("task.check-type.sql-review.description");
    default:
      return undefined;
  }
};

const formatTime = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return "";
  return humanizeTs(
    new Date(Number(timestamp.seconds) * 1000).getTime() / 1000
  );
};

const getCheckResultStatus = (
  status: PlanCheckRun_Result_Status
): "SUCCESS" | "WARNING" | "ERROR" => {
  switch (status) {
    case PlanCheckRun_Result_Status.ERROR:
      return "ERROR";
    case PlanCheckRun_Result_Status.WARNING:
      return "WARNING";
    case PlanCheckRun_Result_Status.SUCCESS:
      return "SUCCESS";
    default:
      return "SUCCESS";
  }
};
</script>
