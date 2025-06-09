<template>
  <div class="px-4 py-3 flex flex-col gap-y-2">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-1">
        <h3 class="text-base font-medium">{{ $t("plan.targets.title") }}</h3>
        <span class="text-control-light">({{ targets.length }})</span>
      </div>
      <!-- TODO(claude): allow to update targets when no rollout is created by plan -->
      <NButton
        v-if="allowEdit"
        size="small"
        @click="showTargetsSelector = true"
      >
        <template #icon>
          <EditIcon class="w-4 h-4" />
        </template>
        {{ $t("common.edit") }}
      </NButton>
    </div>

    <div class="flex-1 overflow-y-auto">
      <div v-if="isLoading" class="flex items-center justify-center py-8">
        <BBSpin />
      </div>
      <div v-else-if="targets.length > 0" class="flex flex-wrap gap-2">
        <NTooltip
          v-for="(item, index) in tableData"
          :key="index"
          placement="top"
        >
          <template #trigger>
            <div
              class="inline-flex items-center gap-x-1.5 px-3 py-1.5 border rounded-lg transition-all cursor-default max-w-[20rem]"
            >
              <component
                :is="item.icon"
                class="w-4 h-4 text-control-light flex-shrink-0"
              />
              <div class="flex items-center gap-x-1 min-w-0">
                <NEllipsis :line-clamp="1" class="text-sm">
                  <span class="font-medium">{{ item.name }}</span>
                  <span
                    v-if="item.type === 'database' && item.instance"
                    class="text-control-light"
                  >
                    <span class="mx-1">·</span>{{ item.instance }}
                  </span>
                </NEllipsis>
              </div>
              <div
                class="text-xs px-2 py-0.5 rounded-md bg-control-bg text-control-light flex-shrink-0"
              >
                {{ getTypeLabel(item.type) }}
              </div>
            </div>
          </template>

          <!-- Tooltip Content -->
          <div class="max-w-sm space-y-1">
            <div class="font-semibold">{{ item.name }}</div>
            <div class="text-sm space-y-0.5">
              <div
                v-if="item.type !== 'instance' && item.instance"
                class="flex items-center gap-1"
              >
                <ServerIcon class="w-3 h-3" />
                <span>{{ item.instance }}</span>
              </div>
              <div v-if="item.environment">
                {{
                  $t("plan.targets.environment", {
                    environment: item.environment,
                  })
                }}
              </div>
              <div v-if="item.description" class="pt-1 text-control-light">
                {{ item.description }}
              </div>
            </div>
          </div>
        </NTooltip>
      </div>
      <div v-else class="text-center text-control-light py-8">
        {{ $t("plan.targets.no-targets-found") }}
      </div>
    </div>

    <TargetsSelectorDrawer
      v-if="project"
      v-model:show="showTargetsSelector"
      :current-targets="targets"
      @confirm="handleUpdateTargets"
    />
  </div>
</template>

<script setup lang="ts">
import {
  ServerIcon,
  DatabaseIcon,
  FolderIcon,
  EditIcon,
} from "lucide-vue-next";
import { NEllipsis, NTooltip, NButton } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import {
  useInstanceV1Store,
  useDatabaseV1Store,
  useDBGroupStore,
  useProjectV1Store,
} from "@/store";
import {
  extractInstanceResourceName,
  instanceV1Name,
  extractDatabaseResourceName,
  extractDatabaseGroupName,
  extractProjectResourceName,
} from "@/utils";
import { usePlanContext } from "../../logic/context";
import { targetsForSpec } from "../../logic/plan";
import TargetsSelectorDrawer from "./TargetsSelectorDrawer.vue";

interface TargetRow {
  target: string;
  type: "instance" | "database" | "databaseGroup";
  icon: any;
  name: string;
  instance?: string;
  environment?: string;
  description?: string;
}

const { t } = useI18n();
const { plan, selectedSpec, isCreating } = usePlanContext();
const instanceStore = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const projectStore = useProjectV1Store();

const isLoading = ref(false);
const showTargetsSelector = ref(false);

const targets = computed(() => {
  if (!selectedSpec.value) return [];
  return targetsForSpec(selectedSpec.value);
});

const isCreateDatabaseSpec = computed(() => {
  return !!selectedSpec.value?.createDatabaseConfig;
});

const project = computed(() => {
  if (!plan.value?.name) return undefined;
  const projectName = `projects/${extractProjectResourceName(plan.value.name)}`;
  return projectStore.getProjectByName(projectName);
});

// Only allow editing in creation mode or if the plan is editable
const allowEdit = computed(() => {
  return isCreating.value && selectedSpec.value;
});

// Prepare data - fetch resources when targets change
watchEffect(async () => {
  isLoading.value = true;
  try {
    if (!selectedSpec.value) return;

    const targetList = targets.value;
    if (targetList.length === 0) return;

    // Prepare promises for fetching resources
    const fetchPromises: Promise<any>[] = [];

    for (const target of targetList) {
      // For create database spec, target is instance
      if (isCreateDatabaseSpec.value) {
        const instanceResourceName = extractInstanceResourceName(target);
        fetchPromises.push(
          instanceStore.getOrFetchInstanceByName(instanceResourceName)
        );
      }
      // For database group targets
      else if (target.includes("/databaseGroups/")) {
        fetchPromises.push(dbGroupStore.getOrFetchDBGroupByName(target));
      }
      // For regular database targets
      else {
        fetchPromises.push(databaseStore.getOrFetchDatabaseByName(target));
      }
    }

    // Fetch all resources in parallel
    await Promise.allSettled(fetchPromises);
  } finally {
    isLoading.value = false;
  }
});

const tableData = computed((): TargetRow[] => {
  if (!selectedSpec.value) return [];

  return targets.value.map((target): TargetRow => {
    // For create database spec, target is instance
    if (isCreateDatabaseSpec.value) {
      const instanceResourceName = extractInstanceResourceName(target);
      const instance = instanceStore.getInstanceByName(instanceResourceName);

      return {
        target,
        type: "instance",
        icon: ServerIcon,
        name: instance ? instanceV1Name(instance) : instanceResourceName,
        environment: instance?.environmentEntity?.title || "Unknown",
        description: instance?.title || "",
      };
    }

    // For database group targets
    if (target.includes("/databaseGroups/")) {
      const groupName = extractDatabaseGroupName(target);
      const dbGroup = dbGroupStore.getDBGroupByName(target);

      return {
        target,
        type: "databaseGroup",
        icon: FolderIcon,
        name: dbGroup?.title || groupName,
      };
    }

    // For regular database targets
    const database = databaseStore.getDatabaseByName(target);

    if (!database) {
      // Fallback when database is not found
      const { instance: instanceId, databaseName } =
        extractDatabaseResourceName(target);
      if (instanceId && databaseName) {
        return {
          target,
          type: "database",
          icon: DatabaseIcon,
          name: databaseName,
          instance: instanceId,
          description: t("plan.targets.database-not-found"),
        };
      }
    }

    const instance = database?.instanceResource;

    return {
      target,
      type: "database",
      icon: DatabaseIcon,
      name: database?.databaseName || target,
      instance: instance ? instanceV1Name(instance) : "",
      environment: database?.effectiveEnvironmentEntity?.title || "",
    };
  });
});

const getTypeLabel = (type: TargetRow["type"]) => {
  const typeLabels = {
    instance: t("plan.targets.type.instance"),
    database: t("plan.targets.type.database"),
    databaseGroup: t("plan.targets.type.database-group"),
  };
  return typeLabels[type];
};

const handleUpdateTargets = (targets: string[]) => {
  if (!selectedSpec.value) return;

  // Update the targets in the spec.
  const config =
    selectedSpec.value.changeDatabaseConfig ||
    selectedSpec.value.exportDataConfig;
  if (config) {
    config.targets = targets;
  }
};
</script>
