<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="$t('database.sync-schema.target-databases')"
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
          databaseListGroupByEnvironment.map((group) => group.environment.uid)
        "
      >
        <NCollapseItem
          v-for="{
            environment,
            databaseList: databaseListInEnvironment,
          } in databaseListGroupByEnvironment"
          :key="environment.uid"
          :name="environment.uid"
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
              <EnvironmentV1Name :environment="environment" :link="false" />
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
              v-for="database in databaseListInEnvironment"
              :key="database.uid"
            >
              <label
                class="border-control-border relative border p-3 flex flex-col gap-y-2 md:flex-row md:pl-4 md:pr-6"
                :class="
                  database.syncState === State.ACTIVE
                    ? 'cursor-pointer'
                    : 'cursor-not-allowed'
                "
              >
                <div class="radio text-sm flex justify-start md:flex-1">
                  <input
                    type="checkbox"
                    class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
                    :checked="isDatabaseSelected(database.uid)"
                    @input="(e: any) => toggleDatabaseSelected(database.uid, e.target.checked)"
                  />
                  <span
                    class="font-medium ml-2 text-main"
                    :class="database.syncState !== State.ACTIVE && 'opacity-40'"
                    >{{ database.databaseName }}</span
                  >
                </div>
                <div
                  class="flex items-center gap-x-1 textinfolabel ml-6 pl-0 md:ml-0 md:pl-0 md:justify-end"
                >
                  <InstanceV1Name
                    :instance="database.instanceEntity"
                    :link="false"
                  />
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
import { computed, reactive } from "vue";
import {
  NCollapse,
  NCollapseItem,
  NButton,
  NDrawer,
  NDrawerContent,
} from "naive-ui";
import { useDatabaseV1Store, useEnvironmentV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import { EnvironmentV1Name, InstanceV1Name } from "@/components/v2";
import { Engine, State } from "@/types/proto/v1/common";

type LocalState = {
  searchText: string;
  selectedDatabaseList: ComposedDatabase[];
};

const props = defineProps<{
  projectId: string;
  engine: Engine;
  selectedDatabaseIdList: string[];
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "update", databaseIdList: string[]): void;
}>();

const environmentV1Store = useEnvironmentV1Store();
const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  searchText: "",
  selectedDatabaseList: props.selectedDatabaseIdList.map((id) => {
    return databaseStore.getDatabaseByUID(id);
  }),
});

const databaseListGroupByEnvironment = computed(() => {
  const databaseList =
    databaseStore.databaseList
      .filter((db) => db.projectEntity.uid === props.projectId)
      .filter((db) => db.databaseName.includes(state.searchText))
      .filter((db) => db.instanceEntity.engine === props.engine) || [];
  const listByEnv = environmentV1Store.environmentList.map((environment) => {
    const list = databaseList.filter(
      (db) => db.instanceEntity.environment === environment.name
    );
    return {
      environment,
      databaseList: list,
    };
  });

  return listByEnv.filter((group) => group.databaseList.length > 0);
});

const isDatabaseSelected = (databaseId: string) => {
  const idList = state.selectedDatabaseList.map((db) => db.uid);
  return idList.includes(databaseId);
};

const toggleDatabaseSelected = (databaseId: string, selected: boolean) => {
  const index = state.selectedDatabaseList.findIndex(
    (db) => db.uid === databaseId
  );
  if (selected) {
    if (index < 0) {
      state.selectedDatabaseList.push(
        databaseStore.getDatabaseByUID(databaseId)
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
  databaseList: ComposedDatabase[],
  on: boolean
) => {
  databaseList
    .filter((db) => db.instanceEntity.environment === environment.name)
    .forEach((db) => toggleDatabaseSelected(db.uid, on));
};

const getAllSelectionStateForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
): { checked: boolean; indeterminate: boolean } => {
  const set = new Set(
    state.selectedDatabaseList
      .filter((db) => db.instanceEntity.environment === environment.name)
      .map((db) => db.uid)
  );
  const checked = databaseList.every((db) => set.has(db.uid));
  const indeterminate = !checked && databaseList.some((db) => set.has(db.uid));

  return {
    checked,
    indeterminate,
  };
};

const getSelectionStateSummaryForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
) => {
  const set = new Set(
    state.selectedDatabaseList
      .filter((db) => db.instanceEntity.environment === environment.name)
      .map((db) => db.uid)
  );
  const selected = databaseList.filter((db) => set.has(db.uid)).length;
  const total = databaseList.length;

  return { selected, total };
};

const handleConfirm = async () => {
  const databaseIdList = state.selectedDatabaseList
    .filter((db) => db.databaseName.includes(state.searchText))
    .map((db) => db.uid);
  emit("update", databaseIdList);
};
</script>
