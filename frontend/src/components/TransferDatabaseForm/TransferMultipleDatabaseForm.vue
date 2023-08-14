<template>
  <DrawerContent :title="$t('quick-action.transfer-in-db-title')">
    <div
      class="px-4 w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)]"
    >
      <slot name="transfer-source-selector" />

      <!-- Leave some margin space to avoid accidentally clicking the Collaspe when trying to click the selector -->
      <NCollapse
        class="mt-6"
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
            <template #item="{ item: database }: { item: ComposedDatabase }">
              <div class="bb-grid-cell gap-x-2 !pl-[23px]">
                <input
                  type="checkbox"
                  class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed focus:ring-accent"
                  :checked="
                    isDatabaseSelectedForEnvironment(
                      database.uid,
                      environment.uid
                    )
                  "
                  @input="(e: any) => toggleDatabaseIdForEnvironment(database.uid, environment.uid, e.target.checked)"
                  @click.stop=""
                />
                <span
                  class="font-medium text-main"
                  :class="database.syncState !== State.ACTIVE && 'opacity-40'"
                  >{{ database.databaseName }}</span
                >
              </div>
              <div v-if="showProjectColumn" class="bb-grid-cell">
                {{ database.projectEntity.title }}
              </div>
              <div class="bb-grid-cell gap-x-1 textinfolabel justify-end">
                <InstanceV1Name
                  :instance="database.instanceEntity"
                  :link="false"
                />
              </div>
            </template>
          </BBGrid>
        </NCollapseItem>
      </NCollapse>
    </div>

    <template #footer>
      <div class="flex-1 flex items-center justify-between">
        <div>
          <div
            v-if="combinedSelectedDatabaseUidList.length > 0"
            class="textinfolabel"
          >
            {{
              $t("database.selected-n-databases", {
                n: combinedSelectedDatabaseUidList.length,
              })
            }}
          </div>
        </div>
        <div class="flex items-center gap-x-3">
          <NButton @click.prevent="$emit('dismiss')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!allowTransfer"
            @click.prevent="transferDatabase"
          >
            {{ $t("common.transfer") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { NButton, NCollapse, NCollapseItem } from "naive-ui";
import { computed, PropType, reactive, watch } from "vue";
import { type BBGridColumn, BBGrid } from "@/bbkit";
import { useDatabaseV1Store, useEnvironmentV1List } from "@/store";
import { ComposedDatabase } from "@/types";
import { State } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";
import {
  DrawerContent,
  InstanceV1Name,
  ProductionEnvironmentV1Icon,
} from "../v2";
import { TransferSource } from "./utils";

type LocalState = {
  selectedDatabaseUidListForEnvironment: Map<string, Set<string>>;
};

const props = defineProps({
  transferSource: {
    type: String as PropType<TransferSource>,
    required: true,
  },
  databaseList: {
    type: Array as PropType<ComposedDatabase[]>,
    default: () => [],
  },
});

const emit = defineEmits<{
  (e: "dismiss"): void;
  (e: "submit", databaseList: ComposedDatabase[]): void;
}>();

const databaseStore = useDatabaseV1Store();
const environmentList = useEnvironmentV1List();

const state = reactive<LocalState>({
  selectedDatabaseUidListForEnvironment: new Map(),
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
    width: "20rem",
  };
  return showProjectColumn.value
    ? [DB_NAME, PROJECT, INSTANCE]
    : [DB_NAME, INSTANCE];
});

const databaseListGroupByEnvironment = computed(() => {
  const listByEnv = environmentList.value.map((environment) => {
    const databaseList = props.databaseList.filter(
      (db) => db.effectiveEnvironment === environment.name
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
    state.selectedDatabaseUidListForEnvironment.clear();
  }
);

const combinedSelectedDatabaseUidList = computed(() => {
  const list: string[] = [];
  for (const listForEnv of state.selectedDatabaseUidListForEnvironment.values()) {
    list.push(...listForEnv);
  }
  return list;
});

const isDatabaseSelectedForEnvironment = (
  databaseId: string,
  environmentId: string
) => {
  const map = state.selectedDatabaseUidListForEnvironment;
  const set = map.get(environmentId) || new Set();
  return set.has(databaseId);
};

const toggleDatabaseIdForEnvironment = (
  databaseId: string,
  environmentId: string,
  selected: boolean
) => {
  const map = state.selectedDatabaseUidListForEnvironment;
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
  databaseList: ComposedDatabase[]
): { checked: boolean; indeterminate: boolean } => {
  const set =
    state.selectedDatabaseUidListForEnvironment.get(environment.uid) ??
    new Set();
  const checked = databaseList.every((db) => set.has(db.uid));
  const indeterminate = !checked && databaseList.some((db) => set.has(db.uid));

  return {
    checked,
    indeterminate,
  };
};

const toggleAllDatabasesSelectionForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[],
  on: boolean
) => {
  databaseList.forEach((db) =>
    toggleDatabaseIdForEnvironment(db.uid, environment.uid, on)
  );
};

const getSelectionStateSummaryForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
) => {
  const set =
    state.selectedDatabaseUidListForEnvironment.get(environment.uid) ||
    new Set();
  const selected = databaseList.filter((db) => set.has(db.uid)).length;
  const total = databaseList.length;

  return { selected, total };
};

const handleClickRow = (db: ComposedDatabase) => {
  const environment = db.effectiveEnvironmentEntity;
  toggleDatabaseIdForEnvironment(
    db.uid,
    environment.uid,
    !isDatabaseSelectedForEnvironment(db.uid, environment.uid)
  );
};

const allowTransfer = computed(
  () => combinedSelectedDatabaseUidList.value.length > 0
);

const transferDatabase = () => {
  if (combinedSelectedDatabaseUidList.value.length === 0) return;

  // If a database can be selected, it must be fetched already.
  // So it's safe that we won't get <<Unknown database>> here.
  const databaseList = combinedSelectedDatabaseUidList.value.map((uid) =>
    databaseStore.getDatabaseByUID(uid)
  );
  emit("submit", databaseList);
};
</script>
