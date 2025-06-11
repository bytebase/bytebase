<template>
  <div class="px-4 py-3 flex flex-col gap-y-2">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-1">
        <h3 class="text-base font-medium">Targets</h3>
        <span class="text-control-light">({{ targets.length }})</span>
      </div>
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
              class="inline-flex items-center gap-x-1.5 px-3 py-1.5 border rounded-lg hover:bg-control-bg-hover hover:border-control-border-hover transition-all cursor-default max-w-[20rem]"
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
                Environment: {{ item.environment }}
              </div>
              <div v-if="item.description" class="pt-1 text-control-light">
                {{ item.description }}
              </div>
            </div>
          </div>
        </NTooltip>
      </div>
      <div v-else class="text-center text-control-light py-8">
        No targets found
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ServerIcon, DatabaseIcon, FolderIcon } from "lucide-vue-next";
import { NEllipsis, NTooltip } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { BBSpin } from "@/bbkit";
import {
  useInstanceV1Store,
  useDatabaseV1Store,
  useDBGroupStore,
} from "@/store";
import { extractInstanceResourceName, instanceV1Name } from "@/utils";
import { targetsForSpec } from "../logic/plan";
import { usePlanSpecContext } from "./SpecDetailView/context";

interface TargetRow {
  target: string;
  type: "instance" | "database" | "databaseGroup";
  icon: any;
  name: string;
  instance?: string;
  environment?: string;
  description?: string;
}

const { selectedSpec } = usePlanSpecContext();
const instanceStore = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();

const isLoading = ref(false);

const targets = computed(() => {
  if (!selectedSpec.value) return [];
  return targetsForSpec(selectedSpec.value);
});

const isCreateDatabaseSpec = computed(() => {
  return !!selectedSpec.value?.createDatabaseConfig;
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
      const match = target.match(/projects\/([^/]+)\/databaseGroups\/([^/]+)/);
      if (match) {
        const [_, projectId, groupName] = match;
        const dbGroup = dbGroupStore.getDBGroupByName(target);

        return {
          target,
          type: "databaseGroup",
          icon: FolderIcon,
          name: dbGroup?.title || groupName,
          description: dbGroup
            ? `Database group in project ${projectId}`
            : target,
        };
      }
    }

    // For regular database targets
    const database = databaseStore.getDatabaseByName(target);

    if (!database) {
      // Fallback when database is not found
      const match = target.match(/instances\/([^/]+)\/databases\/([^/]+)/);
      if (match) {
        const [_, instanceId, databaseName] = match;
        return {
          target,
          type: "database",
          icon: DatabaseIcon,
          name: databaseName,
          instance: instanceId,
          description: "Database not found",
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
      description: database?.labels?.["bb.database.description"] || "",
    };
  });
});

const getTypeLabel = (type: TargetRow["type"]) => {
  const typeLabels = {
    instance: "Instance",
    database: "Database",
    databaseGroup: "Database Group",
  };
  return typeLabels[type];
};
</script>
