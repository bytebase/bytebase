<template>
  <div v-if="show" class="py-3 flex flex-col gap-y-2 overflow-hidden">
    <!-- Row 1: Title + Run button -->
    <div class="flex items-center justify-between gap-2">
      <div class="flex flex-row items-center gap-2">
        <h3 class="text-base">{{ $t("plan.checks.self") }}</h3>
      </div>
      <NButton
        size="tiny"
        :loading="isRunningChecks"
        :disabled="statement.length === 0"
        @click="runChecks"
      >
        <template #icon>
          <PlayIcon class="w-4 h-4" />
        </template>
        {{ $t("common.run") }}
      </NButton>
    </div>

    <!-- Row 2: Status badges -->
    <div class="flex items-center flex-wrap gap-3 text-sm min-w-0">
      <button
        v-if="summary.error > 0"
        class="flex items-center gap-1 hover:opacity-80"
        @click="openDrawer('ERROR')"
      >
        <XCircleIcon class="w-4 h-4 text-error" />
        <span class="text-error">{{ $t("common.error") }}</span>
        <span class="text-error font-medium">{{ summary.error }}</span>
      </button>
      <button
        v-if="summary.warning > 0"
        class="flex items-center gap-1 hover:opacity-80"
        @click="openDrawer('WARNING')"
      >
        <AlertCircleIcon class="w-4 h-4 text-warning" />
        <span class="text-warning">{{ $t("common.warning") }}</span>
        <span class="text-warning font-medium">{{ summary.warning }}</span>
      </button>
      <span
        v-if="
          !isNullOrUndefined(checkResults) &&
          summary.error === 0 &&
          summary.warning === 0
        "
        class="flex items-center gap-1"
      >
        <CheckCircleIcon class="w-4 h-4 text-success" />
        <span class="text-success">{{ $t("common.success") }}</span>
      </span>

      <NTooltip v-if="checkResults && affectedRows > 0">
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

    <!-- Checks Drawer -->
    <Drawer v-model:show="drawerVisible">
      <DrawerContent
        :title="$t('plan.navigator.checks')"
        class="w-[40rem] max-w-[100vw] relative"
      >
        <ChecksView
          :default-status="getDefaultStatus()"
          :plan-check-runs="formattedCheckRuns"
          :is-loading="isRunningChecks"
        />
      </DrawerContent>
    </Drawer>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import {
  AlertCircleIcon,
  CheckCircleIcon,
  CircleQuestionMarkIcon,
  PlayIcon,
  XCircleIcon,
} from "lucide-vue-next";
import { NButton, NTag, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { getLocalSheetByName } from "@/components/Plan";
import Drawer from "@/components/v2/Container/Drawer.vue";
import DrawerContent from "@/components/v2/Container/DrawerContent.vue";
import { releaseServiceClientConnect } from "@/connect";
import { projectNamePrefix } from "@/store";
import {
  type PlanCheckRun,
  type PlanCheckRun_Result,
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
  PlanCheckRun_Status,
  PlanCheckRunSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  CheckReleaseRequestSchema,
  type CheckReleaseResponse_CheckResult,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";
import { type Advice, Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import {
  extractProjectResourceName,
  getSheetStatement,
  isNullOrUndefined,
} from "@/utils";
import { usePlanContext } from "../../logic/context";
import { targetsForSpec } from "../../logic/plan";
import ChecksView from "../ChecksView/ChecksView.vue";
import { useSelectedSpec } from "../SpecDetailView/context";

const { plan } = usePlanContext();
const { selectedSpec } = useSelectedSpec();

const isRunningChecks = ref(false);
const drawerVisible = ref(false);
const selectedStatus = ref<"ERROR" | "WARNING" | "SUCCESS">("ERROR");
const checkResults = ref<CheckReleaseResponse_CheckResult[] | undefined>(
  undefined
);

const statement = computed(() => {
  if (!selectedSpec.value) return "";
  const config =
    selectedSpec.value.config?.case === "changeDatabaseConfig"
      ? selectedSpec.value.config.value
      : undefined;
  if (!config) return "";
  const sheet = getLocalSheetByName(config.sheet);
  return getSheetStatement(sheet);
});

const show = computed(() => {
  if (!selectedSpec.value) {
    return false;
  }
  // Show for change database configs
  return selectedSpec.value.config?.case === "changeDatabaseConfig";
});

// Enhanced advice type with target information
type AdviceWithTarget = Advice & {
  target: string;
};

const allAdvices = computed(() => {
  if (!checkResults.value) {
    return [];
  }
  const advices: AdviceWithTarget[] = [];
  for (const result of checkResults.value) {
    for (const advice of result.advices) {
      advices.push({
        ...advice,
        target: result.target,
      });
    }
  }
  return advices;
});

const summary = computed(() => {
  const result = { success: 0, warning: 0, error: 0 };

  for (const advice of allAdvices.value) {
    if (advice.status === Advice_Level.ERROR) {
      result.error++;
    } else if (advice.status === Advice_Level.WARNING) {
      result.warning++;
    } else if (advice.status === Advice_Level.SUCCESS) {
      result.success++;
    }
  }

  return result;
});

const runChecks = async () => {
  if (!plan.value.name || !selectedSpec.value) return;

  const config =
    selectedSpec.value.config?.case === "changeDatabaseConfig"
      ? selectedSpec.value.config.value
      : undefined;
  if (!config) return;

  isRunningChecks.value = true;
  try {
    // Get the statement from the sheet
    const sheetName = config.sheet;
    const sheet = getLocalSheetByName(sheetName);
    const statement = getSheetStatement(sheet);

    // Get targets
    const targets = targetsForSpec(selectedSpec.value);

    // Run check for all targets
    const request = create(CheckReleaseRequestSchema, {
      parent: `${projectNamePrefix}${extractProjectResourceName(plan.value.name)}`,
      release: {
        type: Release_Type.VERSIONED,
        files: [
          {
            // Use "0" for dummy version.
            version: "0",
            statement: new TextEncoder().encode(statement),
            enableGhost: !config.release && config.enableGhost,
          },
        ],
      },
      targets,
    });
    const response = await releaseServiceClientConnect.checkRelease(request);
    checkResults.value = response.results || [];
  } finally {
    isRunningChecks.value = false;
  }
};

const affectedRows = computed(() => {
  if (!checkResults.value) return 0n;
  return checkResults.value.reduce((acc, result) => {
    return acc + result.affectedRows;
  }, 0n);
});

const openDrawer = (status: "ERROR" | "WARNING" | "SUCCESS") => {
  selectedStatus.value = status;
  drawerVisible.value = true;
};

// Convert string status to Advice_Level
const getDefaultStatus = (): Advice_Level | undefined => {
  switch (selectedStatus.value) {
    case "ERROR":
      return Advice_Level.ERROR;
    case "WARNING":
      return Advice_Level.WARNING;
    case "SUCCESS":
      return Advice_Level.SUCCESS;
    default:
      return undefined;
  }
};

watch(
  () => selectedSpec.value.id,
  () => {
    // Reset drawer and results when spec changes.
    drawerVisible.value = false;
    checkResults.value = undefined;
  },
  {
    immediate: true,
  }
);

// Transform CheckReleaseResponse_CheckResult[] to PlanCheckRun[] format
// With consolidated model, we create a single PlanCheckRun with all results
const transformToFormattedCheckRuns = (
  results: CheckReleaseResponse_CheckResult[]
): PlanCheckRun[] => {
  const allResults: PlanCheckRun_Result[] = [];

  for (const result of results) {
    for (const advice of result.advices) {
      allResults.push(
        create(PlanCheckRun_ResultSchema, {
          status:
            advice.status === Advice_Level.ERROR
              ? Advice_Level.ERROR
              : advice.status === Advice_Level.WARNING
                ? Advice_Level.WARNING
                : Advice_Level.SUCCESS,
          title: advice.title,
          content: advice.content,
          code: advice.code,
          target: result.target,
          type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
          report: {
            case: "sqlReviewReport",
            value: {
              startPosition: advice.startPosition,
            },
          },
        })
      );
    }
  }

  if (allResults.length === 0) {
    return [];
  }

  return [
    create(PlanCheckRunSchema, {
      name: "check-run-0",
      status: PlanCheckRun_Status.DONE,
      results: allResults,
      createTime: { seconds: BigInt(Math.floor(Date.now() / 1000)), nanos: 0 },
    }),
  ];
};

// Format check runs for ChecksView component
const formattedCheckRuns = computed(() => {
  if (!checkResults.value) return [];
  return transformToFormattedCheckRuns(checkResults.value);
});
</script>
