<template>
  <div class="px-4 w-[60rem]">
    <slot name="transfer-source-selector" />

    <NCollapse
      class="overflow-y-auto"
      style="max-height: calc(100vh - 380px)"
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
                  :checked="
                    isDatabaseSelectedForEnvironment(
                      database.id,
                      environment.id
                    )
                  "
                  @input="(e: any) => toggleDatabaseIdForEnvironment(database.id, environment.id, e.target.checked)"
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

    <!-- Update button group -->
    <div class="mt-4 pt-4 border-t border-block-border flex justify-between">
      <div>
        <div
          v-if="combinedSelectedDatabaseIdList.length > 0"
          class="textinfolabel"
        >
          {{
            $t("database.selected-n-databases", {
              n: combinedSelectedDatabaseIdList.length,
            })
          }}
        </div>
      </div>
      <div class="flex items-center">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="$emit('dismiss')"
        >
          {{ $t("common.cancel") }}
        </button>
        <button
          type="button"
          class="btn-primary py-2 px-4 ml-3"
          :disabled="!allowTransfer"
          @click.prevent="transferDatabase"
        >
          {{ $t("common.transfer") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, watch } from "vue";
import { NCollapse, NCollapseItem } from "naive-ui";

import { Database, DatabaseId, Environment, EnvironmentId } from "@/types";
import { TransferSource } from "./utils";
import { useDatabaseStore, useEnvironmentList } from "@/store";

type LocalState = {
  selectedDatabaseIdListForEnvironment: Map<EnvironmentId, Set<DatabaseId>>;
};

const props = defineProps({
  transferSource: {
    type: String as PropType<TransferSource>,
    required: true,
  },
  databaseList: {
    type: Array as PropType<Database[]>,
    default: () => [],
  },
});

const emit = defineEmits<{
  (e: "dismiss"): void;
  (e: "submit", databaseList: Database[]): void;
}>();

const databaseStore = useDatabaseStore();
const environmentList = useEnvironmentList();

const state = reactive<LocalState>({
  selectedDatabaseIdListForEnvironment: new Map(),
});

const databaseListGroupByEnvironment = computed(() => {
  const listByEnv = environmentList.value.map((environment) => {
    const databaseList = props.databaseList.filter(
      (db) => db.instance.environment.id === environment.id
    );
    return {
      environment,
      databaseList,
    };
  });

  return listByEnv.filter((group) => group.databaseList.length > 0);
});

watch(
  () => props.transferSource,
  () => {
    // Clear selected database ID list when transferSource changed.
    state.selectedDatabaseIdListForEnvironment.clear();
  }
);

const combinedSelectedDatabaseIdList = computed(() => {
  const list: DatabaseId[] = [];
  for (const listForEnv of state.selectedDatabaseIdListForEnvironment.values()) {
    list.push(...listForEnv);
  }
  return list;
});

const isDatabaseSelectedForEnvironment = (
  databaseId: DatabaseId,
  environmentId: EnvironmentId
) => {
  const map = state.selectedDatabaseIdListForEnvironment;
  const set = map.get(environmentId) || new Set();
  return set.has(databaseId);
};

const toggleDatabaseIdForEnvironment = (
  databaseId: DatabaseId,
  environmentId: EnvironmentId,
  selected: boolean
) => {
  const map = state.selectedDatabaseIdListForEnvironment;
  const set = map.get(environmentId) || new Set();
  if (selected) {
    set.add(databaseId);
  } else {
    set.delete(databaseId);
  }
  map.set(environmentId, set);
};

const getAllSelectionStateForEnvironment = (
  environment: Environment,
  databaseList: Database[]
): { checked: boolean; indeterminate: boolean } => {
  const set =
    state.selectedDatabaseIdListForEnvironment.get(environment.id) ?? new Set();
  const checked = databaseList.every((db) => set.has(db.id));
  const indeterminate = !checked && databaseList.some((db) => set.has(db.id));

  return {
    checked,
    indeterminate,
  };
};

const toggleAllDatabasesSelectionForEnvironment = (
  environment: Environment,
  databaseList: Database[],
  on: boolean
) => {
  databaseList.forEach((db) =>
    toggleDatabaseIdForEnvironment(db.id, environment.id, on)
  );
};

const getSelectionStateSummaryForEnvironment = (
  environment: Environment,
  databaseList: Database[]
) => {
  const set =
    state.selectedDatabaseIdListForEnvironment.get(environment.id) || new Set();
  const selected = databaseList.filter((db) => set.has(db.id)).length;
  const total = databaseList.length;

  return { selected, total };
};

const allowTransfer = computed(
  () => combinedSelectedDatabaseIdList.value.length > 0
);

const transferDatabase = () => {
  if (combinedSelectedDatabaseIdList.value.length === 0) return;

  // If a database can be selected, it must be fetched already.
  // So it's safe that we won't get <<Unknown database>> here.
  const databaseList = combinedSelectedDatabaseIdList.value.map((id) =>
    databaseStore.getDatabaseById(id)
  );
  emit("submit", databaseList);
};
</script>
