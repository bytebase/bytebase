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
              database.syncState === State.ACTIVE
                ? 'cursor-pointer'
                : 'cursor-not-allowed'
            "
          >
            <div class="radio text-sm flex justify-start md:flex-1">
              <NCheckbox
                :checked="
                  isDatabaseSelectedForEnvironment(
                    database.uid,
                    environment.uid
                  )
                "
                @update:checked="
                  (checked) =>
                    toggleDatabaseUidForEnvironment(
                      database.uid,
                      environment.uid,
                      checked
                    )
                "
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
              <InstanceV1EngineIcon :instance="database.instanceEntity" />
              <span class="flex-1 whitespace-pre-wrap">
                {{ instanceV1Name(database.instanceEntity) }}
              </span>
            </div>
          </label>
        </template>
      </div>
    </NCollapseItem>
  </NCollapse>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import { NCollapse, NCollapseItem, NCheckbox } from "naive-ui";
import { reactive, computed, watch } from "vue";
import { EnvironmentV1Name, InstanceV1EngineIcon } from "@/components/v2";
import { ComposedDatabase } from "@/types";
import { State } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";
import { Project } from "@/types/proto/v1/project_service";
import { instanceV1Name } from "@/utils";

interface LocalState {
  selectedDatabaseUidListForEnvironment: Map<string, string[]>;
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
  selectedDatabaseUidListForEnvironment: new Map<string, string[]>(),
});

watch(
  () => state.selectedDatabaseUidListForEnvironment,
  (selectedDatabaseUidListForEnvironment) => {
    const flattenDatabaseIdList: string[] = [];
    for (const databaseIdList of selectedDatabaseUidListForEnvironment.values()) {
      flattenDatabaseIdList.push(...databaseIdList);
    }
    emit("select-databases", ...flattenDatabaseIdList);
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

const toggleDatabaseUidForEnvironment = (
  databaseId: string,
  environmentId: string,
  selected: boolean
) => {
  const map = state.selectedDatabaseUidListForEnvironment;
  const list = map.get(environmentId) || [];
  const index = list.indexOf(databaseId);
  if (selected) {
    // push the databaseId in if needed
    if (index < 0) {
      list.push(databaseId);
    }
  } else {
    // remove the databaseId if exists
    if (index >= 0) {
      list.splice(index, 1);
    }
  }
  // Set or remove the list to the map
  if (list.length > 0) {
    map.set(environmentId, list);
  } else {
    map.delete(environmentId);
  }
};

const isDatabaseSelectedForEnvironment = (
  databaseId: string,
  environmentId: string
) => {
  const map = state.selectedDatabaseUidListForEnvironment;
  const list = map.get(environmentId) || [];
  return list.includes(databaseId);
};

const getAllSelectionStateForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
): { checked: boolean; indeterminate: boolean } => {
  const set = new Set(
    state.selectedDatabaseUidListForEnvironment.get(environment.uid) ?? []
  );
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
    toggleDatabaseUidForEnvironment(db.uid, environment.uid, on)
  );
};

const getSelectionStateSummaryForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
) => {
  const set = new Set(
    state.selectedDatabaseUidListForEnvironment.get(environment.uid)
  );
  const selected = databaseList.filter((db) => set.has(db.uid)).length;
  const total = databaseList.length;

  return { selected, total };
};
</script>
