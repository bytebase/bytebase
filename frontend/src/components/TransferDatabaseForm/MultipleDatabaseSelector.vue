<template>
  <div class="w-full">
    <!-- Leave some margin space to avoid accidentally clicking the Collaspe when trying to click the selector -->
    <NCollapse
      arrow-placement="left"
      :default-expanded-names="environmentList.map((env) => env.name)"
    >
      <NCollapseItem
        v-for="{
          environment,
          databaseList: databaseListInEnvironment,
        } in databaseListGroupByEnvironment"
        :key="environment.name"
        :name="environment.name"
      >
        <template #header>
          <label class="flex items-center gap-x-2" @click.stop.prevent>
            <NCheckbox
              v-bind="
                getAllSelectionStateForEnvironment(
                  environment,
                  databaseListInEnvironment
                )
              "
              @update:checked="
                toggleAllDatabasesSelectionForEnvironment(
                  environment,
                  databaseListInEnvironment,
                  $event
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
              <div @click.stop.prevent>
                <NCheckbox
                  :checked="
                    isDatabaseSelectedForEnvironment(
                      database.uid,
                      environment.name
                    )
                  "
                  @update:checked="
                    toggleDatabaseIdForEnvironment(
                      database.uid,
                      environment.name,
                      $event
                    )
                  "
                />
              </div>
              <span
                class="font-medium text-main"
                :class="database.syncState !== State.ACTIVE && 'opacity-40'"
              >
                {{ database.databaseName }}
              </span>
            </div>
            <div v-if="showProjectColumn" class="bb-grid-cell">
              <ProjectNameCell :project="database.projectEntity" />
            </div>
            <div class="bb-grid-cell justify-end">
              <InstanceV1Name
                :instance="database.instanceResource"
                :link="false"
              />
            </div>
            <div class="bb-grid-cell textinfolabel justify-end">
              {{ hostPortOfInstanceV1(database.instanceResource) }}
            </div>
          </template>
        </BBGrid>
      </NCollapseItem>
    </NCollapse>
  </div>
</template>

<script lang="ts" setup>
import { NCollapse, NCollapseItem, NCheckbox } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { type BBGridColumn, BBGrid } from "@/bbkit";
import { ProjectNameCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import { useEnvironmentV1List } from "@/store";
import type { ComposedDatabase } from "@/types";
import { State } from "@/types/proto/v1/common";
import type { Environment } from "@/types/proto/v1/environment_service";
import { hostPortOfInstanceV1 } from "@/utils";
import { InstanceV1Name, ProductionEnvironmentV1Icon } from "../v2";
import type { TransferSource } from "./utils";

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
  environmentName: string,
  selected: boolean
) => {
  const map = state.selectedDatabaseUidListForEnvironment;
  const set = map.get(environmentName) || new Set();
  if (selected) {
    set.add(databaseId);
  } else {
    set.delete(databaseId);
  }
  map.set(environmentName, set);
};

watch(
  () => [props.databaseList, props.selectedUidList],
  (args) => {
    const [databaseList, selectedUidList] = args as [
      ComposedDatabase[],
      string[],
    ];
    for (const uid of selectedUidList) {
      const database = databaseList.find((db) => db.uid === uid);
      if (!database) {
        continue;
      }
      toggleDatabaseIdForEnvironment(uid, database.effectiveEnvironment, true);
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
  return [
    {
      width: "1fr",
    },
    {
      width: "8rem",
      hide: !showProjectColumn.value,
    },
    {
      width: "20rem",
    },
    {
      width: "10rem",
    },
  ].filter((col) => !col.hide);
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
  environmentName: string
) => {
  const map = state.selectedDatabaseUidListForEnvironment;
  const set = map.get(environmentName) || new Set();
  return set.has(databaseId);
};

const getAllSelectionStateForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
): { checked: boolean; indeterminate: boolean } => {
  const set =
    state.selectedDatabaseUidListForEnvironment.get(environment.name) ??
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
    toggleDatabaseIdForEnvironment(db.uid, environment.name, on)
  );
};

const getSelectionStateSummaryForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
) => {
  const set =
    state.selectedDatabaseUidListForEnvironment.get(environment.name) ||
    new Set();
  const selected = databaseList.filter((db) => set.has(db.uid)).length;
  const total = databaseList.length;

  return { selected, total };
};

const handleClickRow = (db: ComposedDatabase) => {
  const environmentName = db.effectiveEnvironment;
  toggleDatabaseIdForEnvironment(
    db.uid,
    environmentName,
    !isDatabaseSelectedForEnvironment(db.uid, environmentName)
  );
};
</script>
