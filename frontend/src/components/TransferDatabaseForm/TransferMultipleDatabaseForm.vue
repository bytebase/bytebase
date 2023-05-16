<template>
  <div class="px-4 w-[60rem]">
    <slot name="transfer-source-selector" />

    <NCollapse
      class="overflow-y-auto"
      style="max-height: calc(100vh - 380px)"
      arrow-placement="left"
      :default-expanded-names="environmentList.map((env) => env.uid)"
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
              class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed focus:ring-accent ml-0.5"
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
            <div>{{ environment.title }}</div>
            <ProductionEnvironmentV1Icon :environment="environment" />
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

        <BBGrid
          class="relative bg-white border"
          :column-list="gridColumnList"
          :show-header="false"
          :data-source="databaseListInEnvironment"
          row-key="id"
          @click-row="handleClickRow"
        >
          <template #item="{ item: database }: { item: Database }">
            <div class="bb-grid-cell gap-x-2 !pl-[23px]">
              <input
                type="checkbox"
                class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed focus:ring-accent"
                :checked="
                  isDatabaseSelectedForEnvironment(database.id, environment.uid)
                "
                @input="(e: any) => toggleDatabaseIdForEnvironment(database.id, environment.uid, e.target.checked)"
                @click.stop=""
              />
              <span
                class="font-medium text-main"
                :class="database.syncStatus !== 'OK' && 'opacity-40'"
                >{{ database.name }}</span
              >
            </div>
            <div v-if="showProjectColumn" class="bb-grid-cell">
              {{ database.project.name }}
            </div>
            <div class="bb-grid-cell gap-x-1 textinfolabel justify-end">
              <InstanceEngineIcon :instance="database.instance" />
              <span class="whitespace-pre-wrap">
                {{ instanceName(database.instance) }}
              </span>
            </div>
          </template>
        </BBGrid>
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

import { type BBGridColumn, BBGrid } from "@/bbkit";
import { Database, DatabaseId } from "@/types";
import { TransferSource } from "./utils";
import { useDatabaseStore, useEnvironmentV1List } from "@/store";
import { Environment } from "@/types/proto/v1/environment_service";
import { ProductionEnvironmentV1Icon } from "../v2";

type LocalState = {
  selectedDatabaseIdListForEnvironment: Map<string, Set<DatabaseId>>;
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
const environmentList = useEnvironmentV1List();

const state = reactive<LocalState>({
  selectedDatabaseIdListForEnvironment: new Map(),
});

const showProjectColumn = computed(() => {
  return props.transferSource === "OTHER";
});

const gridColumnList = computed((): BBGridColumn[] => {
  const DB_NAME: BBGridColumn = {
    width: "1fr",
  };
  const PROJECT: BBGridColumn = {
    width: "8rem",
  };
  const INSTANCE: BBGridColumn = {
    width: "14rem",
  };
  return showProjectColumn.value
    ? [DB_NAME, PROJECT, INSTANCE]
    : [DB_NAME, INSTANCE];
});

const databaseListGroupByEnvironment = computed(() => {
  const listByEnv = environmentList.value.map((environment) => {
    const databaseList = props.databaseList.filter(
      (db) => String(db.instance.environment.id) === environment.uid
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
  environmentId: string
) => {
  const map = state.selectedDatabaseIdListForEnvironment;
  const set = map.get(environmentId) || new Set();
  return set.has(databaseId);
};

const toggleDatabaseIdForEnvironment = (
  databaseId: DatabaseId,
  environmentId: string,
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
    state.selectedDatabaseIdListForEnvironment.get(environment.uid) ??
    new Set();
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
    toggleDatabaseIdForEnvironment(db.id, environment.uid, on)
  );
};

const getSelectionStateSummaryForEnvironment = (
  environment: Environment,
  databaseList: Database[]
) => {
  const set =
    state.selectedDatabaseIdListForEnvironment.get(environment.uid) ||
    new Set();
  const selected = databaseList.filter((db) => set.has(db.id)).length;
  const total = databaseList.length;

  return { selected, total };
};

const handleClickRow = (db: Database) => {
  toggleDatabaseIdForEnvironment(
    db.id,
    String(db.instance.environment.id),
    !isDatabaseSelectedForEnvironment(db.id, String(db.instance.environment.id))
  );
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
