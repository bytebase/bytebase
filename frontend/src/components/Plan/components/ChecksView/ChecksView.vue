<template>
  <div class="flex-1 flex flex-col pt-2 pb-4">
    <!-- Header with filters -->
    <div class="flex items-center justify-between px-4">
      <div class="flex items-center gap-4">
        <!-- Check result summary with icons -->
        <div class="flex items-center gap-3">
          <div
            v-if="getChecksCount(PlanCheckRun_Result_Status.ERROR) > 0"
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
              {{ getChecksCount(PlanCheckRun_Result_Status.ERROR) }}
            </span>
          </div>
          <div
            v-if="getChecksCount(PlanCheckRun_Result_Status.WARNING) > 0"
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
              {{ getChecksCount(PlanCheckRun_Result_Status.WARNING) }}
            </span>
          </div>
          <div
            v-if="getChecksCount(PlanCheckRun_Result_Status.SUCCESS) > 0"
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
              {{ getChecksCount(PlanCheckRun_Result_Status.SUCCESS) }}
            </span>
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
          class="px-6 py-4"
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
                <DatabaseDisplay :database="checkRun.target" />
              </div>
            </div>

            <div class="flex items-center gap-2">
              <span class="text-sm text-control">
                {{ formatTime(checkRun.createTime) }}
              </span>
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
  AlertCircleIcon,
  XCircleIcon,
  FileCodeIcon,
  DatabaseIcon,
  ShieldIcon,
  SearchCodeIcon,
} from "lucide-vue-next";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import {
  PlanCheckRun_Result_Status,
  PlanCheckRun_Type,
  type PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import { humanizeTs } from "@/utils";
import { usePlanContext } from "../../logic/context";
import { useResourcePoller } from "../../logic/poller";
import CheckResultItem from "../common/CheckResultItem.vue";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";

const props = defineProps<{
  defaultStatus?: PlanCheckRun_Result_Status;
}>();

const { t } = useI18n();
const { plan, planCheckRuns } = usePlanContext();
const { refreshResources } = useResourcePoller();

const isLoading = ref(true);

const selectedStatus = ref<PlanCheckRun_Result_Status | undefined>(
  props.defaultStatus
);

const hasFilters = computed(() => {
  return selectedStatus.value !== undefined;
});

const filteredCheckRuns = computed(() => {
  return planCheckRuns.value.filter((checkRun) => {
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

watch(
  () => props.defaultStatus,
  async () => {
    isLoading.value = true;
    await refreshResources(["planCheckRuns"], true /** force */);
    isLoading.value = false;
  },
  { immediate: true }
);

const getChecksCount = (status: PlanCheckRun_Result_Status) => {
  return (
    plan.value.planCheckRunStatusCount[PlanCheckRun_Result_Status[status]] || 0
  );
};

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
