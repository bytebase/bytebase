<template>
  <template v-if="state.alterType === 'MULTI_DB'">
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
                      environment.uid
                    )
                  "
                  @input="(e: any) => toggleDatabaseIdForEnvironment(database.id, environment.uid, e.target.checked)"
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
  </template>
  <template v-else>
    <!-- single stage view -->
    <slot name="header"></slot>
    <DatabaseTable
      mode="PROJECT_SHORT"
      table-class="border"
      :custom-click="true"
      :database-list="databaseList"
      @select-database="selectDatabase"
    />
  </template>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import { PropType, computed } from "vue";
import { NCollapse, NCollapseItem } from "naive-ui";

import { Database, DatabaseId } from "../../types";
import { Environment } from "@/types/proto/v1/environment_service";
import { EnvironmentV1Name } from "@/components/v2";
import { Project } from "@/types/proto/v1/project_service";

export type AlterType = "SINGLE_DB" | "MULTI_DB" | "TENANT";

export type State = {
  alterType: AlterType;
  selectedDatabaseIdListForEnvironment: Map<string, DatabaseId[]>;
};

const props = defineProps({
  state: {
    type: Object as PropType<State>,
    required: true,
  },
  project: {
    type: Object as PropType<Project>,
    default: undefined,
  },
  databaseList: {
    type: Array as PropType<Database[]>,
    required: true,
  },
  environmentList: {
    type: Array as PropType<Environment[]>,
    required: true,
  },
});
const emit = defineEmits<{
  (event: "select-database", db: Database): void;
}>();

const databaseListGroupByEnvironment = computed(() => {
  const listByEnv = props.environmentList.map((environment) => {
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

const toggleDatabaseIdForEnvironment = (
  databaseId: DatabaseId,
  environmentId: string,
  selected: boolean
) => {
  const map = props.state.selectedDatabaseIdListForEnvironment;
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
  databaseId: DatabaseId,
  environmentId: string
) => {
  const map = props.state.selectedDatabaseIdListForEnvironment;
  const list = map.get(environmentId) || [];
  return list.includes(databaseId);
};

const getAllSelectionStateForEnvironment = (
  environment: Environment,
  databaseList: Database[]
): { checked: boolean; indeterminate: boolean } => {
  const set = new Set(
    props.state.selectedDatabaseIdListForEnvironment.get(environment.uid) ?? []
  );
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

const selectDatabase = (db: Database) => {
  emit("select-database", db);
};

const getSelectionStateSummaryForEnvironment = (
  environment: Environment,
  databaseList: Database[]
) => {
  const set = new Set(
    props.state.selectedDatabaseIdListForEnvironment.get(environment.uid)
  );
  const selected = databaseList.filter((db) => set.has(db.id)).length;
  const total = databaseList.length;

  return { selected, total };
};
</script>
