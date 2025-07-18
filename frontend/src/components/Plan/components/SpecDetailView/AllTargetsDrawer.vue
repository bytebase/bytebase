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
                  v-for="target in filteredTargets"
                  :key="target"
                  class="inline-flex items-center gap-x-1 px-2 py-1 border rounded-lg transition-all cursor-default"
                >
                  <template v-if="isValidDatabaseName(target)">
                    <DatabaseDisplay :database="target" show-environment />
                  </template>
                  <template v-else-if="isValidDatabaseGroupName(target)">
                    <DatabaseGroupIcon
                      class="w-4 h-4 text-control-light flex-shrink-0"
                    />
                    <DatabaseGroupName
                      :database-group="
                        dbGroupStore.getDBGroupByName(target) as DatabaseGroup
                      "
                      :link="false"
                      :plain="true"
                      class="text-sm"
                    />
                    <NTag size="tiny" round :bordered="false">
                      {{ $t("plan.targets.type.database-group") }}
                    </NTag>
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
import { NTag } from "naive-ui";
import { reactive, computed, watch } from "vue";
import { BBSpin } from "@/bbkit";
import DatabaseGroupIcon from "@/components/DatabaseGroupIcon.vue";
import { Drawer, DrawerContent, SearchBox } from "@/components/v2";
import DatabaseGroupName from "@/components/v2/Model/DatabaseGroupName.vue";
import {
  useDBGroupStore,
  batchGetOrFetchDatabases,
  useDatabaseV1Store,
} from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";

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

const state = reactive<LocalState>({
  searchText: "",
  isLoading: false,
});

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
