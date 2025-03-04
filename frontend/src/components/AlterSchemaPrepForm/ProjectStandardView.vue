<template>
  <!-- multiple stage view -->
  <div class="flex items-center justify-between">
    <div>
      <div v-if="databaseList.length === 0" class="textinfolabel">
        {{ $t("alter-schema.no-databases-in-project") }}
      </div>
    </div>
    <div>
      <slot name="header"></slot>
    </div>
  </div>
  <slot name="sub-header"></slot>

  <NCollapse
    arrow-placement="left"
    :default-expanded-names="
      databaseListGroupByEnvironment.map((group) => group.environment.name)
    "
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
          v-for="(database, dbIndex) in databaseListInEnvironment"
          :key="dbIndex"
        >
          <label
            class="border-control-border relative border p-3 flex flex-col gap-y-2 md:flex-row md:pl-4 md:pr-6"
            :class="
              database.state === State.ACTIVE
                ? 'cursor-pointer'
                : 'cursor-not-allowed'
            "
          >
            <div class="radio text-sm flex justify-start md:flex-1">
              <NCheckbox
                :checked="
                  isDatabaseSelectedForEnvironment(
                    database.name,
                    environment.name
                  )
                "
                @update:checked="
                  (checked) =>
                    toggleDatabaseNameForEnvironment(
                      database.name,
                      environment.name,
                      checked
                    )
                "
              />
              <span
                class="font-medium ml-2 text-main"
                :class="database.state !== State.ACTIVE && 'opacity-40'"
                >{{ database.databaseName }}</span
              >
            </div>
            <div
              class="flex items-center gap-x-2 ml-6 pl-0 md:ml-0 md:pl-0 md:justify-end"
            >
              <InstanceV1Name
                :instance="database.instanceResource"
                :link="false"
              />
              <span class="textinfolabel">
                {{ hostPortOfInstanceV1(database.instanceResource) }}
              </span>
            </div>
          </label>
        </template>
      </div>
    </NCollapseItem>
  </NCollapse>
</template>

<script lang="ts" setup>
import { NCollapse, NCollapseItem, NCheckbox } from "naive-ui";
import { reactive, computed, watch } from "vue";
import { EnvironmentV1Name, InstanceV1Name } from "@/components/v2";
import type { ComposedDatabase } from "@/types";
import { State } from "@/types/proto/v1/common";
import type { Environment } from "@/types/proto/v1/environment_service";
import type { Project } from "@/types/proto/v1/project_service";
import { hostPortOfInstanceV1 } from "@/utils";

interface LocalState {
  selectedDatabaseNameListForEnvironment: Map<string, string[]>;
}

const props = defineProps<{
  project?: Project;
  databaseList: ComposedDatabase[];
  environmentList: Environment[];
}>();

const emit = defineEmits<{
  (event: "select-databases", ...dbList: string[]): void;
}>();

const state = reactive<LocalState>({
  selectedDatabaseNameListForEnvironment: new Map<string, string[]>(),
});

watch(
  () => state.selectedDatabaseNameListForEnvironment,
  (selectedDatabaseNameListForEnvironment) => {
    const flattenDatabaseNameList: string[] = [];
    for (const databaseNameList of selectedDatabaseNameListForEnvironment.values()) {
      flattenDatabaseNameList.push(...databaseNameList);
    }
    emit("select-databases", ...flattenDatabaseNameList);
  },
  { deep: true }
);

const databaseListGroupByEnvironment = computed(() => {
  const listByEnv = props.environmentList.map((environment) => {
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

const toggleDatabaseNameForEnvironment = (
  databaseName: string,
  environmentName: string,
  selected: boolean
) => {
  const map = state.selectedDatabaseNameListForEnvironment;
  const list = map.get(environmentName) || [];
  const index = list.indexOf(databaseName);
  if (selected) {
    // push the databaseId in if needed
    if (index < 0) {
      list.push(databaseName);
    }
  } else {
    // remove the databaseId if exists
    if (index >= 0) {
      list.splice(index, 1);
    }
  }
  // Set or remove the list to the map
  if (list.length > 0) {
    map.set(environmentName, list);
  } else {
    map.delete(environmentName);
  }
};

const isDatabaseSelectedForEnvironment = (
  databaseName: string,
  environmentName: string
) => {
  const map = state.selectedDatabaseNameListForEnvironment;
  const list = map.get(environmentName) || [];
  return list.includes(databaseName);
};

const getAllSelectionStateForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
): { checked: boolean; indeterminate: boolean } => {
  const set = new Set(
    state.selectedDatabaseNameListForEnvironment.get(environment.name) ?? []
  );
  const checked = set.size > 0 && databaseList.every((db) => set.has(db.name));
  const indeterminate = !checked && databaseList.some((db) => set.has(db.name));

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
    toggleDatabaseNameForEnvironment(db.name, environment.name, on)
  );
};

const getSelectionStateSummaryForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
) => {
  const set = new Set(
    state.selectedDatabaseNameListForEnvironment.get(environment.name)
  );
  const selected = databaseList.filter((db) => set.has(db.name)).length;
  const total = databaseList.length;

  return { selected, total };
};
</script>
