<template>
  <Drawer :show="show" @close="$emit('close')">
    <DrawerContent :title="$t('rollout.pending-tasks-preview.title')">
      <div class="w-[400px] space-y-4">
        <p class="text-sm text-control-light">
          {{ $t("rollout.pending-tasks-preview.description") }}
        </p>

        <BBSpin v-if="isLoading" class="mx-auto py-8" />

        <p v-else-if="!pendingTasksByEnv.length" class="py-8 text-center text-control-light">
          {{ $t("rollout.pending-tasks-preview.no-pending-tasks") }}
        </p>

        <div v-for="group in pendingTasksByEnv" :key="group.environment" class="border rounded-lg">
          <div class="flex items-center gap-2 px-3 py-2 bg-gray-50">
            <div class="flex items-center gap-2 flex-1 cursor-pointer" @click="toggleEnv(group.environment)">
              <component
                :is="expandedEnvs.has(group.environment) ? ChevronDownIcon : ChevronRightIcon"
                class="w-4 h-4 text-gray-500"
              />
              <span class="font-medium">{{ extractEnvironmentResourceName(group.environment) }}</span>
              <span class="text-xs text-control-light">
                {{ $t("rollout.pending-tasks-preview.task-count", group.tasks.length) }}
              </span>
            </div>
            <NButton
              size="tiny"
              :loading="creatingEnv === group.environment"
              :disabled="!!creatingEnv"
              @click="handleCreateTasks(group.environment)"
            >
              {{ $t("common.create") }}
            </NButton>
          </div>
          <ul v-if="expandedEnvs.has(group.environment)" class="px-3 py-2 space-y-1">
            <li v-for="task in group.tasks" :key="task.target">
              <DatabaseDisplay :database="task.target" size="small" />
            </li>
          </ul>
        </div>
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { ChevronDownIcon, ChevronRightIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import BBSpin from "@/bbkit/BBSpin.vue";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import { generateRolloutPreview } from "@/components/Plan/logic";
import { Drawer, DrawerContent } from "@/components/v2";
import { rolloutServiceClientConnect } from "@/connect";
import { pushNotification } from "@/store/modules/notification";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractEnvironmentResourceName,
  extractPlanNameFromRolloutName,
} from "@/utils";

interface TaskGroup {
  environment: string;
  tasks: Task[];
}

const props = defineProps<{
  show: boolean;
  plan: Plan;
  rollout: Rollout;
  projectName: string;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "created"): void;
}>();

const { t } = useI18n();
const isLoading = ref(false);
const creatingEnv = ref<string | null>(null);
const pendingTasksByEnv = ref<TaskGroup[]>([]);
const expandedEnvs = ref<Set<string>>(new Set());

const loadPendingTasks = async () => {
  if (pendingTasksByEnv.value.length > 0 || isLoading.value) return;
  isLoading.value = true;
  try {
    const preview = await generateRolloutPreview(props.plan, props.projectName);

    // Build set of existing tasks (target + specId)
    const existingTasks = new Set<string>();
    for (const stage of props.rollout.stages) {
      for (const task of stage.tasks) {
        existingTasks.add(`${task.target}:${task.specId}`);
      }
    }

    // Find uncreated tasks and group by environment
    const groups = new Map<string, Task[]>();
    for (const stage of preview.stages) {
      for (const task of stage.tasks) {
        const key = `${task.target}:${task.specId}`;
        if (!existingTasks.has(key)) {
          let tasks = groups.get(stage.environment);
          if (!tasks) {
            tasks = [];
            groups.set(stage.environment, tasks);
          }
          tasks.push(task);
        }
      }
    }

    pendingTasksByEnv.value = Array.from(groups.entries()).map(
      ([environment, tasks]) => ({ environment, tasks })
    );
    // Auto-expand all groups
    expandedEnvs.value = new Set(groups.keys());
  } finally {
    isLoading.value = false;
  }
};

const handleCreateTasks = async (environment: string) => {
  creatingEnv.value = environment;
  try {
    const request = create(CreateRolloutRequestSchema, {
      parent: extractPlanNameFromRolloutName(props.rollout.name),
      target: environment,
    });
    await rolloutServiceClientConnect.createRollout(request);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.success"),
    });
    // Remove created group from list
    pendingTasksByEnv.value = pendingTasksByEnv.value.filter(
      (g) => g.environment !== environment
    );
    emit("created");
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: String(error),
    });
  } finally {
    creatingEnv.value = null;
  }
};

// Load when drawer opens
watch(
  () => props.show,
  (show) => {
    if (show) {
      loadPendingTasks();
    }
  },
  { immediate: true }
);

const toggleEnv = (env: string) => {
  if (expandedEnvs.value.has(env)) {
    expandedEnvs.value.delete(env);
  } else {
    expandedEnvs.value.add(env);
  }
};
</script>
