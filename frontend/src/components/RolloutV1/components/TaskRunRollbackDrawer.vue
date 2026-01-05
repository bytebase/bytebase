<template>
  <Drawer v-model:show="show" :width="720" placement="right">
    <DrawerContent :title="$t('common.rollback')" :closable="!state.loading">
      <div class="flex flex-col gap-y-4">
        <!-- Steps indicator -->
        <NSteps :current="currentStep" size="small">
          <NStep :title="$t('task.select-task', 2)" />
          <NStep :title="$t('task-run.rollback.preview-statement.self')" />
        </NSteps>

        <!-- Step content -->
        <div class="flex-1">
          <!-- Step 1: Select Tasks -->
          <template v-if="currentStep === 1">
            <TaskTable
              bordered
              :tasks="rollbackableTasks"
              :task-status-filter="[]"
              :selected-tasks="selectedTasks"
              :task-selectable="isTaskSelectable"
              @update:selected-tasks="handleSelectedTasksUpdate"
            />
          </template>

          <!-- Step 2: Preview Statements -->
          <template v-else-if="currentStep === 2">
            <div class="flex flex-col gap-y-4">
              <NAlert type="info">
                {{ $t("task-run.rollback.preview-statement.description") }}
              </NAlert>

              <div class="flex flex-col gap-y-4">
                <div
                  v-for="preview in rollbackPreviews"
                  :key="preview.taskRunName"
                  class="flex flex-col gap-y-2"
                >
                  <DatabaseDisplay :database="preview.task.target" />
                  <template v-if="!isUndefined(preview.statement)">
                    <MonacoEditor
                      v-if="preview.statement"
                      class="border rounded-[3px] text-sm overflow-clip"
                      :content="preview.statement"
                      :readonly="true"
                      :auto-height="{ min: 100, max: 200 }"
                    />
                    <!-- Empty generated statement placeholder -->
                    <div
                      v-else
                      class="flex items-center justify-center p-8 border rounded-[3px] bg-gray-50"
                    >
                      <div class="text-center flex flex-col gap-y-2">
                        <DatabaseBackupIcon
                          class="w-6 h-6 mx-auto text-gray-400"
                        />
                        <p class="text-sm text-gray-500">
                          {{ $t("task-run.rollback.no-statement-generated") }}
                        </p>
                      </div>
                    </div>
                  </template>
                  <div
                    v-else-if="preview.error"
                    class="p-3 border border-red-200 bg-red-50 rounded-sm text-sm text-red-600"
                  >
                    {{ preview.error }}
                  </div>
                  <NSkeleton v-else text :repeat="3" class="h-24" />
                </div>
              </div>
            </div>
          </template>
        </div>
      </div>

      <template #footer>
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-x-2">
            <NButton
              v-if="currentStep === 1"
              quaternary
              :disabled="state.loading"
              @click="handleCancel"
            >
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              v-else
              quaternary
              :disabled="state.loading"
              @click="handlePrevStep"
            >
              {{ $t("common.back") }}
            </NButton>
            <NButton
              v-if="currentStep === 1"
              type="primary"
              :disabled="selectedTasks.length === 0"
              @click="handleNextStep"
            >
              {{ $t("common.next") }}
            </NButton>
            <NButton
              v-else-if="currentStep === 2"
              type="primary"
              :loading="state.loading"
              :disabled="!canCreateRollback"
              @click="handleConfirm"
            >
              {{ $t("common.confirm") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { isUndefined } from "lodash-es";
import { DatabaseBackupIcon } from "lucide-vue-next";
import { NAlert, NButton, NSkeleton, NStep, NSteps } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { MonacoEditor } from "@/components/MonacoEditor";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentProjectV1,
  useSheetV1Store,
} from "@/store";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  CreatePlanRequestSchema,
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type {
  Rollout,
  Task,
  TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import { PreviewTaskRunRollbackRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import {
  databaseForTask,
  extractPlanUID,
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  hasProjectPermissionV2,
} from "@/utils";
import TaskTable from "./TaskTable.vue";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  rollout: Rollout;
  rollbackableTaskRuns: Array<{
    task: Task;
    taskRun: TaskRun;
    database: ReturnType<typeof databaseForTask>;
  }>;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const router = useRouter();
const { project } = useCurrentProjectV1();

const show = defineModel<boolean>("show", { default: false });

const state = reactive<LocalState>({
  loading: false,
});

const currentStep = ref(1);
const selectedTasks = ref<Task[]>([]);
const rollbackPreviews = ref<
  Array<{
    taskRunName: string;
    task: Task;
    database: ReturnType<typeof databaseForTask>;
    statement?: string;
    error?: string;
  }>
>([]);

// Extract just the tasks from rollbackableTaskRuns
const rollbackableTasks = computed(() => {
  return props.rollbackableTaskRuns.map((item) => item.task);
});

// Map selected tasks back to their task runs
const selectedTaskRuns = computed(() => {
  return props.rollbackableTaskRuns.filter((item) =>
    selectedTasks.value.some((task) => task.name === item.task.name)
  );
});

// Task is selectable if it has a corresponding rollbackable task run
const isTaskSelectable = (task: Task): boolean => {
  // In this case, all tasks in rollbackableTasks should be selectable
  // but we can add additional logic here if needed
  return rollbackableTasks.value.some((t) => t.name === task.name);
};

const canCreateRollback = computed(() => {
  return (
    rollbackPreviews.value.length > 0 &&
    rollbackPreviews.value.every((p) => !p.error) &&
    rollbackPreviews.value.some((p) => p.statement) && // At least one non-empty statement
    hasProjectPermissionV2(project.value, "bb.plans.create")
  );
});

const handleSelectedTasksUpdate = (tasks: Task[]) => {
  selectedTasks.value = tasks;
};

const handleCancel = () => {
  show.value = false;
  emit("close");
};

const handleNextStep = async () => {
  if (selectedTasks.value.length === 0) return;

  currentStep.value = 2;
  rollbackPreviews.value = selectedTaskRuns.value.map((item) => ({
    taskRunName: item.taskRun.name,
    task: item.task,
    database: item.database,
  }));

  // Fetch preview for each selected task run
  for (const preview of rollbackPreviews.value) {
    try {
      const request = create(PreviewTaskRunRollbackRequestSchema, {
        name: preview.taskRunName,
      });
      const response = await rolloutServiceClientConnect.previewTaskRunRollback(
        request,
        {
          contextValues: createContextValues().set(silentContextKey, true),
        }
      );
      const index = rollbackPreviews.value.findIndex(
        (p) => p.taskRunName === preview.taskRunName
      );
      if (index !== -1) {
        rollbackPreviews.value[index].statement = response.statement;
      }
    } catch (error) {
      const index = rollbackPreviews.value.findIndex(
        (p) => p.taskRunName === preview.taskRunName
      );
      if (index !== -1) {
        rollbackPreviews.value[index].error = String(error);
      }
    }
  }
};

const handlePrevStep = () => {
  if (currentStep.value === 2) {
    currentStep.value = 1;
  }
};

const handleConfirm = async () => {
  if (!canCreateRollback.value) return;

  state.loading = true;
  try {
    const sheetStore = useSheetV1Store();
    const specs: Plan_Spec[] = [];

    // Create sheets and specs for each rollback
    for (const preview of rollbackPreviews.value) {
      if (!preview.statement) continue;

      // Create a sheet for the rollback statement
      const sheet = await sheetStore.createSheet(
        project.value.name,
        create(SheetSchema, {
          name: `${project.value.name}/sheets/${uuidv4()}`,
          content: new TextEncoder().encode(preview.statement),
        })
      );

      // Create spec for this database
      const spec = create(Plan_SpecSchema, {
        id: uuidv4(),
        config: {
          case: "changeDatabaseConfig",
          value: create(Plan_ChangeDatabaseConfigSchema, {
            targets: [preview.task.target],

            enableGhost: false,
            sheet: sheet.name,
          }),
        },
      });
      specs.push(spec);
    }

    // Create the plan
    const plan = create(PlanSchema, {
      name: `${project.value.name}/plans/${uuidv4()}`,
      title: `Rollback for rollout#${extractPlanUIDFromRolloutName(props.rollout.name)}`,
      description: `This plan is created to rollback ${rollbackPreviews.value.length} task(s) in rollout #${extractPlanUIDFromRolloutName(props.rollout.name)}`,
      specs,
    });

    const request = create(CreatePlanRequestSchema, {
      parent: project.value.name,
      plan,
    });
    const createdPlan = await planServiceClientConnect.createPlan(request);
    router.push({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: {
        projectId: extractProjectResourceName(project.value.name),
        planId: extractPlanUID(createdPlan.name),
      },
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Failed to create rollback plan",
      description: String(error),
    });
  } finally {
    state.loading = false;
  }
};

// Reset state when drawer opens
const resetState = () => {
  currentStep.value = 1;
  selectedTasks.value = [];
  rollbackPreviews.value = [];
  state.loading = false;
};

// Reset state when drawer closes
watch(show, (newValue) => {
  if (newValue) {
    resetState();
    // Auto-select and advance to step 2 if there's only one task
    if (rollbackableTasks.value.length === 1) {
      selectedTasks.value = [rollbackableTasks.value[0]];
      // Use nextTick to ensure state is properly initialized before advancing
      nextTick(() => {
        handleNextStep();
      });
    }
  }
});
</script>
