<template>
  <div class="flex-1 flex flex-col">
    <!-- Header with status counts -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <div class="flex items-center gap-3" v-if="hasAnyStatus">
          <div
            v-if="statusSummary.error > 0"
            class="flex items-center gap-1 text-error cursor-pointer"
            :class="getItemClass(Advice_Level.ERROR)"
            @click="toggleSelectedStatus(Advice_Level.ERROR)"
          >
            <XCircleIcon class="w-5 h-5" />
            <span>{{ $t("common.error") }}</span>
            <span>{{ statusSummary.error }}</span>
          </div>
          <div
            v-if="statusSummary.warning > 0"
            class="flex items-center gap-1 text-warning cursor-pointer"
            :class="getItemClass(Advice_Level.WARNING)"
            @click="toggleSelectedStatus(Advice_Level.WARNING)"
          >
            <AlertCircleIcon class="w-5 h-5" />
            <span>{{ $t("common.warning") }}</span>
            <span>{{ statusSummary.warning }}</span>
          </div>
          <div
            v-if="statusSummary.success > 0"
            class="flex items-center gap-1 text-success cursor-pointer"
            :class="getItemClass(Advice_Level.SUCCESS)"
            @click="toggleSelectedStatus(Advice_Level.SUCCESS)"
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
        v-else-if="filteredResultGroups.length === 0"
        class="flex flex-col items-center justify-center py-12"
      >
        <CheckCircleIcon class="w-12 h-12 text-control-light opacity-50 mb-4" />
        <div class="text-lg text-control-light">
          {{
            hasFilters
              ? $t("plan.checks.no-results-match-filters")
              : $t("plan.checks.no-check-results")
          }}
        </div>
      </div>
      <div v-else>
        <div class="divide-y">
          <!-- Group by result type/target -->
          <div
            v-for="group in displayedResultGroups"
            :key="group.key"
            class="px-2 py-4"
          >
          <!-- Check Run Header -->
          <div class="flex flex-wrap items-start justify-between gap-2 mb-2">
            <div class="flex items-center gap-3 shrink-0">
              <component
                :is="getCheckTypeIcon(group.type)"
                class="w-5 h-5 text-control-light"
              />
              <div class="flex flex-row items-center gap-2">
                <span class="text-sm font-medium">
                  {{ getCheckTypeLabel(group.type) }}
                </span>
                <NTooltip v-if="getCheckTypeDescription(group.type)">
                  <template #trigger>
                    <CircleQuestionMarkIcon
                      class="w-4 h-4 text-control-light"
                    />
                  </template>
                  {{ getCheckTypeDescription(group.type) }}
                </NTooltip>
                <span v-if="group.createTime" class="text-sm text-control-light">
                  {{ formatTime(group.createTime) }}
                </span>
              </div>
            </div>

            <div class="flex items-center gap-2 min-w-0 max-w-[50%]">
              <DatabaseDisplay :database="group.target" />
            </div>
          </div>

          <!-- Results for this group -->
          <div class="flex flex-col gap-y-2 pl-8">
            <CheckResultItem
              v-for="(result, idx) in group.results"
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
        <!-- Load more button -->
        <div v-if="hasMore" class="flex justify-center py-4">
          <button
            class="text-sm text-accent hover:underline"
            @click="loadMore"
          >
            {{ $t("common.load-more") }} ({{ remainingCount }})
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import {
  AlertCircleIcon,
  CheckCircleIcon,
  CircleQuestionMarkIcon,
  FileCodeIcon,
  SearchCodeIcon,
  ShieldIcon,
  XCircleIcon,
} from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import {
  type PlanCheckRun,
  type PlanCheckRun_Result,
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
  PlanCheckRun_Status,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { humanizeTs } from "@/utils";
import CheckResultItem from "../common/CheckResultItem.vue";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";

interface ResultGroup {
  key: string;
  type: PlanCheckRun_Result_Type;
  target: string;
  createTime?: Timestamp;
  results: PlanCheckRun_Result[];
}

const props = defineProps<{
  defaultStatus?: Advice_Level;
  planCheckRuns: PlanCheckRun[];
  isLoading?: boolean;
}>();

const { t } = useI18n();

const selectedStatus = ref<Advice_Level | undefined>(props.defaultStatus);

// Pagination - load more
const PAGE_SIZE = 10;
const displayCount = ref(PAGE_SIZE);

const loadMore = () => {
  displayCount.value += PAGE_SIZE;
};

const hasFilters = computed(() => {
  return selectedStatus.value !== undefined;
});

// Calculate status summary from plan check runs
const statusSummary = computed(() => {
  const summary = { error: 0, warning: 0, success: 0 };

  for (const checkRun of props.planCheckRuns) {
    // Count failed plan check runs as errors
    if (checkRun.status === PlanCheckRun_Status.FAILED) {
      summary.error++;
    }

    for (const result of checkRun.results) {
      if (result.status === Advice_Level.ERROR) {
        summary.error++;
      } else if (result.status === Advice_Level.WARNING) {
        summary.warning++;
      } else if (result.status === Advice_Level.SUCCESS) {
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

const toggleSelectedStatus = (status: Advice_Level) => {
  if (selectedStatus.value === status) {
    selectedStatus.value = undefined; // Deselect if already selected
  } else {
    selectedStatus.value = status; // Select the new status
  }
  // Reset pagination when filter changes
  displayCount.value = PAGE_SIZE;
};

const getItemClass = (status: Advice_Level) => {
  return selectedStatus.value === status
    ? "bg-gray-100 rounded-lg px-2 py-1"
    : "px-2 py-1";
};

// Group results by type and target
const filteredResultGroups = computed(() => {
  const groups: ResultGroup[] = [];
  const groupMap = new Map<string, ResultGroup>();

  for (const checkRun of props.planCheckRuns) {
    // Handle failed check runs
    if (checkRun.status === PlanCheckRun_Status.FAILED) {
      if (
        selectedStatus.value === undefined ||
        selectedStatus.value === Advice_Level.ERROR
      ) {
        // Create a synthetic error result for the failed run
        const syntheticResult = create(PlanCheckRun_ResultSchema, {
          status: Advice_Level.ERROR,
          title: "Check Failed",
          content: checkRun.error || "Plan check run failed",
          code: 0,
        });
        const key = `failed-${checkRun.name}`;
        groups.push({
          key,
          type: PlanCheckRun_Result_Type.TYPE_UNSPECIFIED,
          target: "",
          createTime: checkRun.createTime,
          results: [syntheticResult],
        });
      }
    }

    // Process results
    for (const result of checkRun.results) {
      // Filter by status
      if (
        selectedStatus.value !== undefined &&
        result.status !== selectedStatus.value
      ) {
        continue;
      }

      const key = `${result.type}-${result.target}`;
      let group = groupMap.get(key);
      if (!group) {
        group = {
          key,
          type: result.type,
          target: result.target,
          createTime: checkRun.createTime,
          results: [],
        };
        groupMap.set(key, group);
        groups.push(group);
      }
      group.results.push(result);
    }
  }

  return groups;
});

// Paginated groups
const displayedResultGroups = computed(() => {
  return filteredResultGroups.value.slice(0, displayCount.value);
});

const hasMore = computed(() => {
  return filteredResultGroups.value.length > displayCount.value;
});

const remainingCount = computed(() => {
  return filteredResultGroups.value.length - displayCount.value;
});

// Check type configuration lookup
const checkTypeConfig: Partial<
  Record<
    PlanCheckRun_Result_Type,
    { icon: typeof FileCodeIcon; labelKey: string }
  >
> = {
  [PlanCheckRun_Result_Type.STATEMENT_ADVISE]: {
    icon: SearchCodeIcon,
    labelKey: "task.check-type.sql-review.self",
  },
  [PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT]: {
    icon: FileCodeIcon,
    labelKey: "task.check-type.summary-report",
  },
  [PlanCheckRun_Result_Type.GHOST_SYNC]: {
    icon: ShieldIcon,
    labelKey: "task.check-type.ghost-sync",
  },
};

const getCheckTypeIcon = (type: PlanCheckRun_Result_Type) => {
  return checkTypeConfig[type]?.icon ?? FileCodeIcon;
};

const getCheckTypeLabel = (type: PlanCheckRun_Result_Type) => {
  const key = checkTypeConfig[type]?.labelKey;
  return key ? t(key) : type.toString();
};

const getCheckTypeDescription = (type: PlanCheckRun_Result_Type) => {
  switch (type) {
    case PlanCheckRun_Result_Type.STATEMENT_ADVISE:
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

const adviceLevelToStatus: Partial<
  Record<Advice_Level, "SUCCESS" | "WARNING" | "ERROR">
> = {
  [Advice_Level.ERROR]: "ERROR",
  [Advice_Level.WARNING]: "WARNING",
  [Advice_Level.SUCCESS]: "SUCCESS",
};

const getCheckResultStatus = (
  status: Advice_Level
): "SUCCESS" | "WARNING" | "ERROR" => {
  return adviceLevelToStatus[status] ?? "SUCCESS";
};
</script>
