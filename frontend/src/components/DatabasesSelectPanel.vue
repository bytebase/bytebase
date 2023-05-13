<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="$t('common.database')"
      :closable="true"
      class="w-[30rem] max-w-[100vw] relative"
    >
      <div class="flex items-center justify-end mx-2 mb-2">
        <BBTableSearch
          class="m-px"
          :placeholder="$t('database.search-database')"
          @change-text="(text: string) => (state.searchText = text)"
        />
      </div>
      <NCollapse
        class="overflow-y-auto"
        arrow-placement="left"
        :default-expanded-names="
          databaseListGroupByEnvironment.map((group) => group.environment.id)
        "
      >
        <NCollapseItem
          v-for="{
            environment,
            databaseList: databaseListInEnvironment,
          } in databaseListGroupByEnvironment"
          :key="environment.id"
          :name="environment.id"
        >
          <template #header>
            <label class="flex items-center gap-x-2" @click.stop="">
              <input
                type="checkbox"
                class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent ml-0.5"
                v-bind="
                  getAllSelectionStateForEnvironment(
                    environment,
                    databaseListInEnvironment
                  )
                "
                @click.stop=""
                @input="
                  toggleAllDatabasesSelectionForEnvironment(
                    environment,
                    databaseListInEnvironment,
                    ($event.target as HTMLInputElement).checked
                  )
                "
              />
              <div>{{ environment.name }}</div>
              <ProductionEnvironmentIcon
                class="w-4 h-4 -ml-1"
                :environment="environment"
              />
            </label>
          </template>

          <template #header-extra>
            <div class="flex items-center text-xs text-gray-500 mr-2">
              {{
                $t(
                  "database.n-selected-m-in-total",
                  getSelectionStateSummaryForEnvironment(
                    environment,
                    databaseListInEnvironment
                  )
                )
              }}
            </div>
          </template>

          <div class="relative bg-white rounded-md -space-y-px px-2">
            <template
              v-for="(database, dbIndex) in databaseListInEnvironment"
              :key="dbIndex"
            >
              <label
                class="border-control-border relative border p-3 flex flex-col gap-y-2 md:flex-row md:pl-4 md:pr-6"
                :class="
                  database.syncStatus == 'OK'
                    ? 'cursor-pointer'
                    : 'cursor-not-allowed'
                "
              >
                <div class="radio text-sm flex justify-start md:flex-1">
                  <input
                    type="checkbox"
                    class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
                    :checked="isDatabaseSelected(database.id)"
                    @input="(e: any) => toggleDatabaseSelected(database.id, e.target.checked)"
                  />
                  <span
                    class="font-medium ml-2 text-main"
                    :class="database.syncStatus !== 'OK' && 'opacity-40'"
                    >{{ database.name }}</span
                  >
                </div>
                <div
                  class="flex items-center gap-x-1 textinfolabel ml-6 pl-0 md:ml-0 md:pl-0 md:justify-end"
                >
                  <InstanceEngineIcon :instance="database.instance" />
                  <span class="flex-1 whitespace-pre-wrap">
                    {{ instanceName(database.instance) }}
                  </span>
                </div>
              </label>
            </template>
          </div>
        </NCollapseItem>
      </NCollapse>

      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary" @click="handleConfirm">
            {{ $t("common.select") }}
          </NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script setup lang="ts">
import {
  NCollapse,
  NCollapseItem,
  NButton,
  NDrawer,
  NDrawerContent,
} from "naive-ui";
import { computed, reactive, PropType } from "vue";
import { useDatabaseStore, useEnvironmentStore } from "@/store";
import { Database, DatabaseId, Environment, ProjectId } from "@/types";

type LocalState = {
  searchText: string;
  selectedDatabaseList: Database[];
};

const props = defineProps({
  projectId: {
    type: Number as PropType<ProjectId>,
    required: true,
  },
  selectedDatabaseIdList: {
    type: Array as PropType<DatabaseId[]>,
    required: true,
  },
});

const emit = defineEmits<{
  (event: "close"): void;
  (event: "update", databaseIdList: DatabaseId[]): void;
}>();

const environmentStore = useEnvironmentStore();
const databaseStore = useDatabaseStore();
const state = reactive<LocalState>({
  searchText: "",
  selectedDatabaseList: props.selectedDatabaseIdList.map((id) => {
    return databaseStore.getDatabaseById(id);
  }),
});

const databaseListGroupByEnvironment = computed(() => {
  const databaseList =
    databaseStore.databaseListByProjectId
      .get(props.projectId)
      ?.filter((db) => db.name.includes(state.searchText)) || [];
  const listByEnv = environmentStore.environmentList.map((environment) => {
    const list = databaseList.filter(
      (db) => db.instance.environment.id === environment.id
    );
    return {
      environment,
      databaseList: list,
    };
  });

  return listByEnv.filter((group) => group.databaseList.length > 0);
});

const isDatabaseSelected = (databaseId: DatabaseId) => {
  const idList = state.selectedDatabaseList.map((db) => db.id);
  return idList.includes(databaseId);
};

const toggleDatabaseSelected = (databaseId: DatabaseId, selected: boolean) => {
  const index = state.selectedDatabaseList.findIndex(
    (db) => db.id === databaseId
  );
  if (selected) {
    if (index < 0) {
      state.selectedDatabaseList.push(
        databaseStore.getDatabaseById(databaseId)
      );
    }
  } else {
    if (index >= 0) {
      state.selectedDatabaseList.splice(index, 1);
    }
  }
};

const toggleAllDatabasesSelectionForEnvironment = (
  environment: Environment,
  databaseList: Database[],
  on: boolean
) => {
  databaseList
    .filter((db) => db.instance.environment.id === environment.id)
    .forEach((db) => toggleDatabaseSelected(db.id, on));
};

const getAllSelectionStateForEnvironment = (
  environment: Environment,
  databaseList: Database[]
): { checked: boolean; indeterminate: boolean } => {
  const set = new Set(
    state.selectedDatabaseList
      .filter((db) => db.instance.environment.id === environment.id)
      .map((db) => db.id)
  );
  const checked = databaseList.every((db) => set.has(db.id));
  const indeterminate = !checked && databaseList.some((db) => set.has(db.id));

  return {
    checked,
    indeterminate,
  };
};

const getSelectionStateSummaryForEnvironment = (
  environment: Environment,
  databaseList: Database[]
) => {
  const set = new Set(
    state.selectedDatabaseList
      .filter((db) => db.instance.environment.id === environment.id)
      .map((db) => db.id)
  );
  const selected = databaseList.filter((db) => set.has(db.id)).length;
  const total = databaseList.length;

  return { selected, total };
};

const handleConfirm = async () => {
  const databaseIdList = state.selectedDatabaseList
    .filter((db) => db.name.includes(state.searchText))
    .map((db) => db.id);
  emit("update", databaseIdList);
};
</script>
