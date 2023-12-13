<template>
  <div class="w-full">
    <!-- Leave some margin space to avoid accidentally clicking the Collaspe when trying to click the selector -->
    <NCollapse
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
</template>

<script lang="ts" setup>
import { NCollapse, NCollapseItem } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { type BBGridColumn, BBGrid } from "@/bbkit";
import { useEnvironmentV1List } from "@/store";
import { ComposedDatabase } from "@/types";
import { State } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";
import { InstanceV1Name, ProductionEnvironmentV1Icon } from "../v2";
import { TransferSource } from "./utils";

type LocalState = {
  selectedDatabaseUidListForEnvironment: Map<string, Set<string>>;
};

const props = withDefaults(
  defineProps<{
    transferSource: TransferSource;
    databaseList: ComposedDatabase[];
    selectedUidList?: string[];
  }>(),
  {
    selectedUidList: () => [],
  }
);

const emit = defineEmits<{
  (e: "update:selectedUidList", uidList: string[]): void;
}>();

const environmentList = useEnvironmentV1List();

const state = reactive<LocalState>({
  selectedDatabaseUidListForEnvironment: new Map(),
});

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

watch(
  () => [props.databaseList, props.selectedUidList],
  (args) => {
    const [databaseList, selectedUidList] = args as [
      ComposedDatabase[],
      string[]
    ];
    for (const uid of selectedUidList) {
      const database = databaseList.find((db) => db.uid === uid);
      if (!database) {
        continue;
      }
      toggleDatabaseIdForEnvironment(
        uid,
        database.effectiveEnvironmentEntity.uid,
        true
      );
    }
  },
  { immediate: true }
);

watch(
  () => state.selectedDatabaseUidListForEnvironment,
  (map) => {
    const list: string[] = [];
    for (const listForEnv of map.values()) {
      list.push(...listForEnv);
    }
    emit("update:selectedUidList", list);
  },
  { deep: true }
);

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

const isDatabaseSelectedForEnvironment = (
  databaseId: string,
  environmentId: string
) => {
  const map = state.selectedDatabaseUidListForEnvironment;
  const set = map.get(environmentId) || new Set();
  return set.has(databaseId);
};

const getAllSelectionStateForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
): { checked: boolean; indeterminate: boolean } => {
  const set =
    state.selectedDatabaseUidListForEnvironment.get(environment.uid) ??
    new Set();
  const checked = set.size > 0 && databaseList.every((db) => set.has(db.uid));
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
</script>
