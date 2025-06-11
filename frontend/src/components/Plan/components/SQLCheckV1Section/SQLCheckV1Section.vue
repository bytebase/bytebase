<template>
  <div v-if="show" class="px-4 pt-3 flex flex-col gap-y-1 overflow-hidden">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-1">
        <h3 class="text-base font-medium">{{ $t("plan.checks.self") }}</h3>
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
        :title="drawerTitle"
        class="w-[40rem] max-w-[100vw] relative"
      >
        <div class="w-full h-full flex flex-col">
          <!-- Drawer Header -->
          <div class="flex items-center justify-between px-2 py-2 border-b">
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
          <div class="flex-1 overflow-y-auto py-2">
            <div v-if="drawerAdvices.length > 0" class="space-y-2">
              <div
                v-for="(advice, idx) in drawerAdvices"
                :key="idx"
                class="space-y-1 p-3 border rounded-lg bg-gray-50"
              >
                <div class="flex items-start justify-between">
                  <div class="text-sm font-medium text-main">
                    {{ getAdviceTitle(advice) }}
                  </div>
                  <component
                    :is="getStatusIcon(advice.status)"
                    class="w-4 h-4 flex-shrink-0"
                    :class="getStatusColor(advice.status)"
                  />
                </div>

                <!-- Target Database -->
                <div class="text-xs text-control font-medium">
                  Database: {{ formatTarget(advice.target) }}
                </div>

                <!-- Advice Content -->
                <div v-if="advice.content" class="text-xs text-control-light">
                  {{ advice.content }}
                </div>

                <!-- Location Info -->
                <div
                  v-if="advice.startPosition && advice.startPosition.line > 0"
                  class="text-xs text-control-lighter"
                >
                  Line {{ advice.startPosition.line }}, Column
                  {{ advice.startPosition.column }}
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
import {
  CheckCircleIcon,
  AlertCircleIcon,
  XCircleIcon,
  PlayIcon,
} from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref, watch } from "vue";
import { getLocalSheetByName } from "@/components/Plan";
import Drawer from "@/components/v2/Container/Drawer.vue";
import DrawerContent from "@/components/v2/Container/DrawerContent.vue";
import { releaseServiceClient } from "@/grpcweb";
import { getRuleLocalization, ruleTemplateMapV2 } from "@/types";
import { Plan_ChangeDatabaseConfig_Type } from "@/types/proto/v1/plan_service";
import {
  ReleaseFileType,
  Release_File_ChangeType,
  type CheckReleaseResponse_CheckResult,
} from "@/types/proto/v1/release_service";
import { Advice_Status, type Advice } from "@/types/proto/v1/sql_service";
import {
  getSheetStatement,
  isNullOrUndefined,
  extractDatabaseResourceName,
} from "@/utils";
import { usePlanContext } from "../../logic/context";
import { targetsForSpec } from "../../logic/plan";
import { usePlanSpecContext } from "../SpecDetailView/context";

const { plan } = usePlanContext();
const { selectedSpec } = usePlanSpecContext();

const isRunningChecks = ref(false);
const drawerVisible = ref(false);
const selectedStatus = ref<"ERROR" | "WARNING" | "SUCCESS">("ERROR");
const checkResults = ref<CheckReleaseResponse_CheckResult[] | undefined>(
  undefined
);

const statement = computed(() => {
  if (!selectedSpec.value) return "";
  const config = selectedSpec.value.changeDatabaseConfig;
  if (!config) return "";
  const sheet = getLocalSheetByName(config.sheet);
  return getSheetStatement(sheet);
});

const show = computed(() => {
  if (!selectedSpec.value) {
    return false;
  }
  // Show for change database configs
  return !!selectedSpec.value.changeDatabaseConfig;
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

const drawerTitle = computed(() => {
  if (selectedStatus.value === "ERROR") return "Error Details";
  if (selectedStatus.value === "WARNING") return "Warning Details";
  return "Success Details";
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

  const config = selectedSpec.value.changeDatabaseConfig;
  if (!config) return;

  isRunningChecks.value = true;
  try {
    // Get the statement from the sheet
    const sheetName = config.sheet;
    const sheet = getLocalSheetByName(sheetName);
    const statement = getSheetStatement(sheet);

    // Get targets
    const targets = targetsForSpec(selectedSpec.value);

    // Get project from plan name
    const projectName = `projects/${plan.value.name.split("/")[1]}`;

    // Run check for all targets
    const response = await releaseServiceClient.checkRelease({
      parent: projectName,
      release: {
        files: [
          {
            // Use "0" for dummy version.
            version: "0",
            type: ReleaseFileType.VERSIONED,
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

    checkResults.value = response.results || [];
  } finally {
    isRunningChecks.value = false;
  }
};

const openDrawer = (status: "ERROR" | "WARNING" | "SUCCESS") => {
  selectedStatus.value = status;
  drawerVisible.value = true;
};

const getStatusIcon = (status: Advice_Status | string) => {
  if (status === Advice_Status.ERROR || status === "ERROR") return XCircleIcon;
  if (status === Advice_Status.WARNING || status === "WARNING")
    return AlertCircleIcon;
  return CheckCircleIcon;
};

const getStatusColor = (status: Advice_Status | string) => {
  if (status === Advice_Status.ERROR || status === "ERROR") return "text-error";
  if (status === Advice_Status.WARNING || status === "WARNING")
    return "text-warning";
  return "text-success";
};

const formatTarget = (target: string): string => {
  const { instanceName, databaseName } = extractDatabaseResourceName(target);
  if (instanceName && databaseName) {
    return `${databaseName} (${instanceName})`;
  }
  return target;
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

const getAdviceTitle = (advice: Advice): string => {
  let title = advice.title;
  const rule = getRuleTemplateByType(advice.title);
  if (rule) {
    const ruleLocalization = getRuleLocalization(rule.type, rule.engine);
    if (ruleLocalization.title) {
      title = ruleLocalization.title;
    }
  }
  return title;
};

const getRuleTemplateByType = (type: string) => {
  for (const mapByType of ruleTemplateMapV2.values()) {
    if (mapByType.has(type)) {
      return mapByType.get(type);
    }
  }
  return;
};
</script>
