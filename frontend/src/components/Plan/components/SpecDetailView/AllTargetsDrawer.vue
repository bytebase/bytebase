<template>
  <Drawer :show="show" width="auto" @update:show="$emit('update:show', $event)">
    <DrawerContent
      :title="`${$t('plan.targets.title')} (${targets.length})`"
      :closable="true"
      class="w-[50rem] max-w-[100vw] relative"
    >
      <template #default>
        <div class="w-full h-full flex flex-col gap-y-4">
          <!-- Search bar -->
          <div class="px-4">
            <SearchBox
              v-model:value="state.searchText"
              :placeholder="$t('common.search')"
            />
          </div>

          <!-- Targets list -->
          <div class="flex-1 px-4 pb-4 overflow-hidden">
            <div
              v-if="state.isLoading"
              class="flex items-center justify-center h-full"
            >
              <BBSpin />
            </div>
            <div
              v-else-if="filteredTargets.length > 0"
              class="h-full overflow-y-auto"
            >
              <div class="flex flex-wrap gap-2">
                <div
                  v-for="item in filteredTargets"
                  :key="item.target"
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
                  <span class="text-sm text-gray-500" v-if="item.environment">
                    ({{ item.environment }})
                  </span>
                  <div class="flex items-center gap-x-1 min-w-0 text-sm">
                    <NEllipsis :line-clamp="1">
                      <span class="font-medium">{{ item.name }}</span>
                    </NEllipsis>
                  </div>
                  <div
                    class="text-xs px-2 py-0.5 rounded-md bg-control-bg text-control-light flex-shrink-0"
                  >
                    {{ getTypeLabel(item.type) }}
                  </div>
                </div>
              </div>
            </div>
            <div
              v-else
              class="flex items-center justify-center h-full text-control-light"
            >
              {{ $t("common.no-data") }}
            </div>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { ServerIcon, DatabaseIcon, FolderIcon } from "lucide-vue-next";
import { NEllipsis } from "naive-ui";
import { reactive, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import EngineIcon from "@/components/Icon/EngineIcon.vue";
import { Drawer, DrawerContent, SearchBox } from "@/components/v2";
import {
  useInstanceV1Store,
  useDatabaseV1Store,
  useDBGroupStore,
  batchGetOrFetchDatabases,
} from "@/store";
import {
  isValidDatabaseGroupName,
  isValidDatabaseName,
  isValidInstanceName,
} from "@/types";
import type { Engine } from "@/types/proto/v1/common";
import {
  extractInstanceResourceName,
  instanceV1Name,
  extractDatabaseResourceName,
  extractDatabaseGroupName,
} from "@/utils";

interface Props {
  show: boolean;
  targets: string[];
}

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

interface LocalState {
  searchText: string;
  isLoading: boolean;
  targetRows: TargetRow[];
}

const props = defineProps<Props>();
defineEmits<{
  "update:show": [show: boolean];
}>();

const { t } = useI18n();
const instanceStore = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();

const state = reactive<LocalState>({
  searchText: "",
  isLoading: false,
  targetRows: [],
});

const filteredTargets = computed(() => {
  if (!state.searchText) {
    return state.targetRows;
  }
  const searchText = state.searchText.toLowerCase();
  return state.targetRows.filter((row) => {
    return (
      row.name.toLowerCase().includes(searchText) ||
      row.instance?.toLowerCase().includes(searchText) ||
      row.environment?.toLowerCase().includes(searchText)
    );
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

const loadAllTargets = async () => {
  if (props.targets.length === 0) {
    state.targetRows = [];
    return;
  }

  state.isLoading = true;

  try {
    // Separate different types of targets for optimized fetching
    const databaseTargets: string[] = [];
    const instanceTargets: string[] = [];
    const dbGroupTargets: string[] = [];

    for (const target of props.targets) {
      if (isValidDatabaseGroupName(target)) {
        // If target is a valid database group name
        dbGroupTargets.push(target);
      } else if (isValidDatabaseName(target)) {
        databaseTargets.push(target);
      } else if (isValidInstanceName(target)) {
        instanceTargets.push(target);
      }
    }

    // Batch fetch all targets
    const fetchPromises: Promise<any>[] = [];

    if (databaseTargets.length > 0) {
      fetchPromises.push(batchGetOrFetchDatabases(databaseTargets));
    }

    const instancePromises = instanceTargets.map((target) => {
      const instanceResourceName = extractInstanceResourceName(target);
      return instanceStore.getOrFetchInstanceByName(instanceResourceName);
    });
    fetchPromises.push(...instancePromises);

    const dbGroupPromises = dbGroupTargets.map((target) =>
      dbGroupStore.getOrFetchDBGroupByName(target)
    );
    fetchPromises.push(...dbGroupPromises);

    await Promise.allSettled(fetchPromises);

    // Build target rows
    const rows: TargetRow[] = props.targets.map((target): TargetRow => {
      if (!isValidDatabaseName(target) && isValidInstanceName(target)) {
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
      if (isValidDatabaseGroupName(target)) {
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

    state.targetRows = rows;
  } catch (error) {
    console.error("Failed to load targets:", error);
  } finally {
    state.isLoading = false;
  }
};

// Load all targets when drawer opens
watch(
  () => props.show,
  async (show) => {
    if (show) {
      state.searchText = "";
      await loadAllTargets();
    }
  }
);

// Reload if targets change while drawer is open
watch(
  () => props.targets,
  async () => {
    if (props.show) {
      await loadAllTargets();
    }
  }
);
</script>
