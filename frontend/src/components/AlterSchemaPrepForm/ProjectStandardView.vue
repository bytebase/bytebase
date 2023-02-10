<template>
  <template v-if="state.alterType === 'MULTI_DB'">
    <!-- multiple stage view -->
    <div v-if="databaseList.length === 0" class="textinfolabel">
      {{ $t("alter-schema.no-databases-in-project") }}
    </div>
    <template v-else>
      <slot name="header"></slot>
    </template>

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

<script lang="ts">
/* eslint-disable vue/no-mutating-props */
import { defineComponent, PropType, computed } from "vue";
import { NCollapse, NCollapseItem } from "naive-ui";

import {
  Database,
  DatabaseId,
  Environment,
  EnvironmentId,
  Project,
} from "../../types";

export type AlterType = "SINGLE_DB" | "MULTI_DB" | "TENANT";

export type State = {
  alterType: AlterType;
  selectedDatabaseIdListForEnvironment: Map<EnvironmentId, DatabaseId[]>;
};

export default defineComponent({
  name: "ProjectStandardView",
  components: {
    NCollapse,
    NCollapseItem,
  },
  props: {
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
  },
  emits: ["select-database"],
  setup(props, { emit }) {
    const databaseListGroupByEnvironment = computed(() => {
      const listByEnv = props.environmentList.map((environment) => {
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

    const toggleDatabaseIdForEnvironment = (
      databaseId: DatabaseId,
      environmentId: EnvironmentId,
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
      environmentId: EnvironmentId
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
        props.state.selectedDatabaseIdListForEnvironment.get(environment.id) ??
          []
      );
      const checked = databaseList.every((db) => set.has(db.id));
      const indeterminate =
        !checked && databaseList.some((db) => set.has(db.id));

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

    const selectDatabase = (db: Database) => {
      emit("select-database", db);
    };

    const getSelectionStateSummaryForEnvironment = (
      environment: Environment,
      databaseList: Database[]
    ) => {
      const set = new Set(
        props.state.selectedDatabaseIdListForEnvironment.get(environment.id)
      );
      const selected = databaseList.filter((db) => set.has(db.id)).length;
      const total = databaseList.length;

      return { selected, total };
    };

    return {
      databaseListGroupByEnvironment,
      toggleDatabaseIdForEnvironment,
      isDatabaseSelectedForEnvironment,
      getAllSelectionStateForEnvironment,
      toggleAllDatabasesSelectionForEnvironment,
      selectDatabase,
      getSelectionStateSummaryForEnvironment,
    };
  },
});
</script>
