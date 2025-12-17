<template>
  <Drawer :show="show" width="auto" @update:show="$emit('update:show', $event)">
    <DrawerContent
      :title="`${$t('plan.targets.title')} (${targets.length})`"
      :closable="true"
      class="w-200 max-w-[100vw] relative"
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
                  v-for="target in filteredTargets"
                  :key="target"
                  class="inline-flex items-center gap-x-1 px-2 py-1 border rounded-lg transition-all cursor-default"
                >
                  <template v-if="isValidDatabaseName(target)">
                    <DatabaseDisplay :database="target" show-environment />
                  </template>
                  <template v-else-if="isValidDatabaseGroupName(target)">
                    <DatabaseGroupTargetDisplay :target="target" />
                  </template>
                  <template v-else>
                    <!-- Unknown resource -->
                    <span class="text-sm">{{ target }}</span>
                  </template>
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
import { computed, reactive, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { Drawer, DrawerContent, SearchBox } from "@/components/v2";
import {
  batchGetOrFetchDatabases,
  useDatabaseV1Store,
  useDBGroupStore,
} from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { usePlanContext } from "../../logic/context";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";
import DatabaseGroupTargetDisplay from "./DatabaseGroupTargetDisplay.vue";

interface Props {
  show: boolean;
  targets: string[];
}

interface LocalState {
  searchText: string;
  isLoading: boolean;
}

const props = defineProps<Props>();
defineEmits<{
  "update:show": [show: boolean];
}>();

const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const { plan } = usePlanContext();

const state = reactive<LocalState>({
  searchText: "",
  isLoading: false,
});

// Get databases from deployment mapping for a database group
const getDatabasesForGroup = (groupName: string): string[] => {
  const deployment = plan.value.deployment;
  if (!deployment) return [];
  const mapping = deployment.databaseGroupMappings.find(
    (m: { databaseGroup: string }) => m.databaseGroup === groupName
  );
  return mapping?.databases ?? [];
};

const filteredTargets = computed(() => {
  if (!state.searchText) {
    return props.targets;
  }

  const searchText = state.searchText.toLowerCase();
  return props.targets.filter((target: string) => {
    if (isValidDatabaseName(target)) {
      const db = databaseStore.getDatabaseByName(target);
      return db.databaseName.toLowerCase().includes(searchText);
    }
    return (target as string).toLocaleLowerCase().includes(searchText);
  });
});

const loadAllTargets = async () => {
  if (props.targets.length === 0) {
    return;
  }

  state.isLoading = true;

  try {
    // Separate different types of targets for optimized fetching
    const databaseTargets: string[] = [];
    const dbGroupTargets: string[] = [];

    for (const target of props.targets) {
      if (isValidDatabaseGroupName(target)) {
        // If target is a valid database group name
        dbGroupTargets.push(target);
        // Also collect databases from deployment mapping for this group
        const mappedDatabases = getDatabasesForGroup(target);
        databaseTargets.push(...mappedDatabases);
      } else if (isValidDatabaseName(target)) {
        databaseTargets.push(target);
      }
    }

    // Batch fetch all targets
    const fetchPromises: Promise<any>[] = [];

    if (databaseTargets.length > 0) {
      fetchPromises.push(batchGetOrFetchDatabases(databaseTargets));
    }

    const dbGroupPromises = dbGroupTargets.map((target) =>
      dbGroupStore.getOrFetchDBGroupByName(target)
    );
    fetchPromises.push(...dbGroupPromises);

    await Promise.allSettled(fetchPromises);
  } catch (error) {
    console.error("Failed to load targets:", error);
  } finally {
    state.isLoading = false;
  }
};

watch(
  () => props.show,
  async (show) => {
    if (show) {
      state.searchText = "";
    }
  },
);

// Load all targets when drawer opens
watch(
  () => props.show,
  async (show) => {
    if (show) {
      await loadAllTargets();
    }
  },
  { once: true }
);
</script>
