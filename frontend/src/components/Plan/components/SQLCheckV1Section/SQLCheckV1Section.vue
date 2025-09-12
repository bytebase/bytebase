<template>
  <div v-if="show" class="pt-3 flex flex-col gap-y-1 overflow-hidden">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <h3 class="text-base font-medium">{{ $t("plan.checks.self") }}</h3>

        <NTooltip v-if="checkResults && affectedRows > 0">
          <template #trigger>
            <NTag round :bordered="false">
              <span class="text-sm text-control-light mr-1">{{
                $t("task.check-type.affected-rows.self")
              }}</span>
              <span class="text-sm font-medium">
                {{ affectedRows }}
              </span>
            </NTag>
          </template>
          {{ $t("task.check-type.affected-rows.description") }}
        </NTooltip>
      </div>

      <div class="flex items-center gap-4">
        <!-- Status Summary -->
        <div class="flex items-center gap-2 text-sm">
          <button
            v-if="summary.error > 0"
            class="flex items-center gap-1 hover:opacity-80"
            @click="openDrawer('ERROR')"
          >
            <XCircleIcon class="w-5 h-5 text-error" />
            <span class="text-error font-medium">{{ summary.error }}</span>
          </button>
          <button
            v-if="summary.warning > 0"
            class="flex items-center gap-1 hover:opacity-80"
            @click="openDrawer('WARNING')"
          >
            <AlertCircleIcon class="w-5 h-5 text-warning" />
            <span class="text-warning font-medium">{{ summary.warning }}</span>
          </button>
          <span
            v-if="
              !isNullOrUndefined(checkResults) &&
              summary.error === 0 &&
              summary.warning === 0
            "
            class="flex items-center"
          >
            <CheckCircleIcon class="w-5 h-5 text-success" />
          </span>
        </div>

        <!-- Run Checks Button -->
        <NButton
          size="small"
          :loading="isRunningChecks"
          :disabled="statement.length === 0"
          @click="runChecks"
        >
          <template #icon>
            <PlayIcon class="w-4 h-4" />
          </template>
          {{ $t("task.run-checks") }}
        </NButton>
      </div>
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
  CheckCircleIcon,
  AlertCircleIcon,
  XCircleIcon,
  PlayIcon,
} from "lucide-vue-next";
import { NButton, NTag, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { getLocalSheetByName } from "@/components/Plan";
import Drawer from "@/components/v2/Container/Drawer.vue";
import DrawerContent from "@/components/v2/Container/DrawerContent.vue";
import { releaseServiceClientConnect } from "@/grpcweb";
import { projectNamePrefix } from "@/store";
import {
  Plan_ChangeDatabaseConfig_Type,
  PlanCheckRun_Type,
  PlanCheckRun_Status,
  PlanCheckRun_Result_Status,
  type PlanCheckRun,
  type PlanCheckRun_Result,
  PlanCheckRunSchema,
  PlanCheckRun_ResultSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  CheckReleaseRequestSchema,
  Release_File_Type,
} from "@/types/proto-es/v1/release_service_pb";
import {
  Release_File_ChangeType,
  type CheckReleaseResponse_CheckResult,
} from "@/types/proto-es/v1/release_service_pb";
import { Advice_Status, type Advice } from "@/types/proto-es/v1/sql_service_pb";
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
const selectedSpec = useSelectedSpec();

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
    if (advice.status === Advice_Status.ERROR) {
      result.error++;
    } else if (advice.status === Advice_Status.WARNING) {
      result.warning++;
    } else if (advice.status === Advice_Status.SUCCESS) {
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
        files: [
          {
            // Use "0" for dummy version.
            version: "0",
            type: Release_File_Type.VERSIONED,
            statement: new TextEncoder().encode(statement),
            changeType:
              config.type === Plan_ChangeDatabaseConfig_Type.DATA
                ? Release_File_ChangeType.DML
                : Release_File_ChangeType.DDL,
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

// Convert string status to PlanCheckRun_Result_Status
const getDefaultStatus = (): PlanCheckRun_Result_Status | undefined => {
  switch (selectedStatus.value) {
    case "ERROR":
      return PlanCheckRun_Result_Status.ERROR;
    case "WARNING":
      return PlanCheckRun_Result_Status.WARNING;
    case "SUCCESS":
      return PlanCheckRun_Result_Status.SUCCESS;
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
const transformToFormattedCheckRuns = (
  results: CheckReleaseResponse_CheckResult[]
): PlanCheckRun[] => {
  return results.map((result, index) => {
    const planCheckRunResults: PlanCheckRun_Result[] = result.advices.map(
      (advice) => {
        return create(PlanCheckRun_ResultSchema, {
          status:
            advice.status === Advice_Status.ERROR
              ? PlanCheckRun_Result_Status.ERROR
              : advice.status === Advice_Status.WARNING
                ? PlanCheckRun_Result_Status.WARNING
                : PlanCheckRun_Result_Status.SUCCESS,
          title: advice.title,
          content: advice.content,
          code: advice.code,
          report: {
            case: "sqlReviewReport",
            value: {
              startPosition: advice.startPosition,
              line: advice.startPosition ? advice.startPosition.line : 0,
            },
          },
        });
      }
    );

    return create(PlanCheckRunSchema, {
      name: `check-run-${index}`,
      type: PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE,
      status: PlanCheckRun_Status.DONE,
      target: result.target,
      results: planCheckRunResults,
      createTime: { seconds: BigInt(Math.floor(Date.now() / 1000)), nanos: 0 },
    });
  });
};

// Format check runs for ChecksView component
const formattedCheckRuns = computed(() => {
  if (!checkResults.value) return [];
  return transformToFormattedCheckRuns(checkResults.value);
});
</script>
