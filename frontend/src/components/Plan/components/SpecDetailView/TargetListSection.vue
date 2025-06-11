<template>
  <div class="px-4 flex flex-col gap-y-2">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-1">
        <h3 class="text-base font-medium">{{ $t("plan.targets.title") }}</h3>
        <span class="text-control-light">({{ targets.length }})</span>
      </div>
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

    <div class="relative flex-1">
      <div
        v-if="!paginationState.initialized && paginationState.isRequesting"
        class="flex items-center justify-center py-8"
      >
        <BBSpin />
      </div>
      <div
        v-else-if="targets.length > 0"
        ref="targetContainer"
        class="flex flex-wrap gap-2 overflow-y-auto"
        :style="{
          'max-height': `${MAX_LIST_HEIGHT}px`,
        }"
      >
        <NTooltip
          v-for="(item, index) in tableData"
          :key="index"
          placement="top"
        >
          <template #trigger>
            <div
              class="inline-flex items-center gap-x-1.5 px-3 py-1.5 border rounded-lg transition-all cursor-default max-w-[20rem]"
            >
              <EngineIcon
                v-if="item.type === 'database' && item.engine"
                :engine="item.engine"
                custom-class="w-4 h-4 text-control-light flex-shrink-0"
              />
              <component
                v-else
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

        <!-- Load More Button -->
        <div
          v-if="targets.length > paginationState.index"
          class="w-full flex items-center justify-end"
        >
          <NButton
            size="small"
            quaternary
            :loading="paginationState.isRequesting"
            @click="loadNextPage"
          >
            {{ $t("common.load-more") }}
          </NButton>
        </div>
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
import { useDebounceFn } from "@vueuse/core";
import {
  ServerIcon,
  DatabaseIcon,
  FolderIcon,
  EditIcon,
} from "lucide-vue-next";
import { NEllipsis, NTooltip, NButton } from "naive-ui";
import { computed, ref, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import EngineIcon from "@/components/Icon/EngineIcon.vue";
import { planServiceClient } from "@/grpcweb";
import {
  useInstanceV1Store,
  useDatabaseV1Store,
  useDBGroupStore,
  useProjectV1Store,
  pushNotification,
  batchGetOrFetchDatabases,
} from "@/store";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import type { Engine } from "@/types/proto/v1/common";
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
import { usePlanSpecContext } from "./context";

interface TargetRow {
  target: string;
  type: "instance" | "database" | "databaseGroup";
  icon: any;
  name: string;
  instance?: string;
  environment?: string;
  description?: string;
  engine?: Engine;
}

interface PaginationState {
  // Index is the current number of targets to show.
  index: number;
  initialized: boolean;
  isRequesting: boolean;
}

interface LocalState {
  pageStatePerSpec: Map<string, PaginationState>;
}

const MAX_LIST_HEIGHT = 256;
const TARGETS_PER_PAGE = 16;

const { t } = useI18n();
const { plan, isCreating, events } = usePlanContext();
const { selectedSpec } = usePlanSpecContext();
const instanceStore = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const projectStore = useProjectV1Store();

const state = reactive<LocalState>({
  pageStatePerSpec: new Map<string, PaginationState>(),
});

const showTargetsSelector = ref(false);
const targetContainer = ref<HTMLDivElement>();

const targets = computed(() => {
  if (!selectedSpec.value) return [];
  return targetsForSpec(selectedSpec.value);
});

const isCreateDatabaseSpec = computed(() => {
  return !!selectedSpec.value?.createDatabaseConfig;
});

// Create a unique key for the spec based on its properties
const specKey = computed(() => {
  if (!selectedSpec.value) return "";
  return JSON.stringify({
    createDb: !!selectedSpec.value.createDatabaseConfig,
    changeDb: !!selectedSpec.value.changeDatabaseConfig,
    exportData: !!selectedSpec.value.exportDataConfig,
    id: selectedSpec.value.id,
  });
});

const paginationState = computed(
  () =>
    state.pageStatePerSpec.get(specKey.value) ?? {
      index: 0,
      initialized: false,
      isRequesting: false,
    }
);

const updatePaginationState = (patch: Partial<PaginationState>) => {
  if (!selectedSpec.value) return;
  state.pageStatePerSpec.set(specKey.value, {
    ...paginationState.value,
    ...patch,
  });
};

const project = computed(() => {
  if (!plan.value?.name) return undefined;
  const projectName = `projects/${extractProjectResourceName(plan.value.name)}`;
  return projectStore.getProjectByName(projectName);
});

// Only allow editing in creation mode or if the plan is editable.
// An empty string for `plan.value.rollout` indicates that the plan is in a draft or uninitialized state,
// which allows edits to be made.
const allowEdit = computed(() => {
  return (isCreating.value || plan.value.rollout === "") && selectedSpec.value;
});

const loadMore = useDebounceFn(async () => {
  const fromIndex = paginationState.value.index;
  const toIndex = fromIndex + TARGETS_PER_PAGE;
  const targetList = targets.value.slice(fromIndex, toIndex);

  if (targetList.length === 0) {
    updatePaginationState({
      index: toIndex,
      initialized: true,
    });
    return;
  }

  // Separate different types of targets for optimized fetching
  const databaseTargets: string[] = [];
  const instanceTargets: string[] = [];
  const dbGroupTargets: string[] = [];

  for (const target of targetList) {
    if (isCreateDatabaseSpec.value) {
      instanceTargets.push(target);
    } else if (target.includes("/databaseGroups/")) {
      dbGroupTargets.push(target);
    } else {
      databaseTargets.push(target);
    }
  }

  try {
    // Use BatchGetDatabases for database targets
    if (databaseTargets.length > 0) {
      await batchGetOrFetchDatabases(databaseTargets);
    }

    // Fetch instances for create database specs
    const instancePromises = instanceTargets.map((target) => {
      const instanceResourceName = extractInstanceResourceName(target);
      return instanceStore.getOrFetchInstanceByName(instanceResourceName);
    });

    // Fetch database groups
    const dbGroupPromises = dbGroupTargets.map((target) =>
      dbGroupStore.getOrFetchDBGroupByName(target)
    );

    // Wait for all remaining promises
    await Promise.allSettled([...instancePromises, ...dbGroupPromises]);
  } catch {
    // Ignore errors - some targets might not be found
  } finally {
    updatePaginationState({
      index: toIndex,
      initialized: true,
    });
  }
}, DEBOUNCE_SEARCH_DELAY);

const loadNextPage = async () => {
  if (paginationState.value.isRequesting) {
    return;
  }
  updatePaginationState({
    isRequesting: true,
  });
  try {
    await loadMore();
  } catch {
    // Ignore errors
  } finally {
    updatePaginationState({
      isRequesting: false,
    });
  }
};

const tableData = computed((): TargetRow[] => {
  if (!selectedSpec.value) return [];

  // Only show targets up to the current pagination index
  const visibleTargets = targets.value.slice(0, paginationState.value.index);

  return visibleTargets.map((target): TargetRow => {
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
      engine: instance?.engine,
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

const handleUpdateTargets = async (targets: string[]) => {
  if (!selectedSpec.value) return;

  // Update the targets in the spec.
  const config =
    selectedSpec.value.changeDatabaseConfig ||
    selectedSpec.value.exportDataConfig;
  if (config) {
    config.targets = targets;
  }

  if (!isCreating.value) {
    await planServiceClient.updatePlan({
      plan: plan.value,
      updateMask: ["specs"],
    });
    events.emit("status-changed", {
      eager: true,
    });
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
};

// Initialize pagination when spec changes
watch(
  specKey,
  async () => {
    if (!paginationState.value.initialized && targets.value.length > 0) {
      await loadNextPage();
    }
  },
  { immediate: true }
);
</script>
