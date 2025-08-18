<template>
  <div v-if="show" class="px-4 pt-3 flex flex-col gap-y-1 overflow-hidden">
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

    <!-- Status Drawer -->
    <Drawer v-model:show="drawerVisible">
      <DrawerContent
        :title="$t('plan.navigator.checks')"
        class="w-[40rem] max-w-[100vw] relative"
      >
        <div class="w-full h-full flex flex-col">
          <!-- Status Tabs Header -->
          <div class="flex items-center gap-3">
            <div
              v-if="summary.error > 0"
              class="flex items-center gap-1 px-2 py-1 cursor-pointer"
              :class="[
                selectedStatus === 'ERROR' && 'bg-gray-100 rounded-lg',
                'text-lg text-error',
              ]"
              @click="selectedStatus = 'ERROR'"
            >
              <XCircleIcon class="w-6 h-6" />
              <span>{{ $t("common.error") }}</span>
              <span class="font-semibold">{{ summary.error }}</span>
            </div>
            <div
              v-if="summary.warning > 0"
              class="flex items-center gap-1 px-2 py-1 cursor-pointer"
              :class="[
                selectedStatus === 'WARNING' && 'bg-gray-100 rounded-lg',
                'text-lg text-warning',
              ]"
              @click="selectedStatus = 'WARNING'"
            >
              <AlertCircleIcon class="w-6 h-6" />
              <span>{{ $t("common.warning") }}</span>
              <span class="font-semibold">{{ summary.warning }}</span>
            </div>
            <div
              v-if="summary.success > 0"
              class="flex items-center gap-1 px-2 py-1 cursor-pointer"
              :class="[
                selectedStatus === 'SUCCESS' && 'bg-gray-100 rounded-lg',
                'text-lg text-success',
              ]"
              @click="selectedStatus = 'SUCCESS'"
            >
              <CheckCircleIcon class="w-6 h-6" />
              <span>{{ $t("common.success") }}</span>
              <span class="font-semibold">{{ summary.success }}</span>
            </div>
          </div>

          <!-- Drawer Content -->
          <div class="flex-1 overflow-y-auto py-4">
            <div v-if="drawerAdvices.length > 0" class="space-y-2">
              <CheckResultItem
                v-for="(advice, idx) in drawerAdvices"
                :key="idx"
                :status="getCheckResultStatus(advice.status)"
                :title="advice.title"
                :content="advice.content"
                :position="advice.startPosition"
                :report-type="'sqlReviewReport'"
              />
            </div>
            <div v-else class="text-center py-8 text-control-light">
              {{ $t("common.no-data") }}
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
} from "lucide-vue-next";
import { NButton, NTag, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { getLocalSheetByName } from "@/components/Plan";
import Drawer from "@/components/v2/Container/Drawer.vue";
import DrawerContent from "@/components/v2/Container/DrawerContent.vue";
import { releaseServiceClientConnect } from "@/grpcweb";
import { projectNamePrefix } from "@/store";
import { Plan_ChangeDatabaseConfig_Type } from "@/types/proto-es/v1/plan_service_pb";
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
import { useSelectedSpec } from "../SpecDetailView/context";
import CheckResultItem from "../common/CheckResultItem.vue";

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

const drawerAdvices = computed(() => {
  return allAdvices.value.filter((advice) => {
    if (selectedStatus.value === "ERROR") {
      return advice.status === Advice_Status.ERROR;
    } else if (selectedStatus.value === "WARNING") {
      return advice.status === Advice_Status.WARNING;
    } else {
      return advice.status === Advice_Status.SUCCESS;
    }
  });
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

const getCheckResultStatus = (
  status: Advice_Status
): "SUCCESS" | "WARNING" | "ERROR" => {
  switch (status) {
    case Advice_Status.ERROR:
      return "ERROR";
    case Advice_Status.WARNING:
      return "WARNING";
    case Advice_Status.SUCCESS:
      return "SUCCESS";
    default:
      return "SUCCESS";
  }
};
</script>
