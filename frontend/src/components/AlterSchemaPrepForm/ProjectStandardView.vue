<template>
  <!-- eslint-disable vue/no-mutating-props -->

  <template v-if="state.alterType === 'MULTI_DB'">
    <!-- multiple stage view -->
    <div class="textinfolabel">
      {{ $t("alter-schema.alter-multiple-db-info") }}
    </div>
    <slot name="header"></slot>
    <div class="space-y-4">
      <div
        v-for="{
          environment,
          databaseList: databaseListInEnvironment,
        } in databaseListGroupByEnvironment"
        :key="environment.id"
      >
        <label class="flex items-center gap-x-2 mb-2 mt-4">
          <input
            type="checkbox"
            class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent ml-[calc(1rem+1px)]"
            v-bind="
              getAllSelectionStateForEnvironment(
                environment,
                databaseListInEnvironment
              )
            "
            @input="
              toggleAllDatabasesSelectionForEnvironment(
                environment,
                databaseListInEnvironment,
                ($event.target as HTMLInputElement).checked
              )
            "
          />
          <div>{{ environment.name }}</div>
          <ProtectedEnvironmentIcon
            class="w-4 h-4 -ml-1"
            :environment="environment"
          />
        </label>
        <div class="relative bg-white rounded-md -space-y-px">
          <template
            v-for="(database, dbIndex) in databaseListInEnvironment"
            :key="dbIndex"
          >
            <label
              class="border-control-border relative border p-3 flex flex-col md:pl-4 md:pr-6 md:grid md:grid-cols-2"
              :class="
                database.syncStatus == 'OK'
                  ? 'cursor-pointer'
                  : 'cursor-not-allowed'
              "
            >
              <div class="radio text-sm">
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
                  class="font-medium"
                  :class="
                    database.syncStatus == 'OK'
                      ? 'ml-2 text-main'
                      : 'ml-6 text-control-light'
                  "
                  >{{ database.name }}</span
                >
              </div>
              <p
                class="textinfolabel ml-6 pl-1 text-sm md:ml-0 md:pl-0 md:text-right"
              >
                {{ $t("database.last-sync-status") }}:
                <span
                  :class="
                    database.syncStatus == 'OK'
                      ? 'textlabel'
                      : 'text-sm font-medium text-error'
                  "
                  >{{ database.syncStatus }}</span
                >
              </p>
            </label>
          </template>
        </div>
      </div>
    </div>
  </template>
  <template v-else>
    <!-- single stage view -->
    <slot name="header"></slot>
    <DatabaseTable
      mode="PROJECT_SHORT"
      :bordered="true"
      :custom-click="true"
      :database-list="databaseList"
      @select-database="selectDatabase"
    />
  </template>
</template>

<script lang="ts">
/* eslint-disable vue/no-mutating-props */
import { defineComponent, watch, PropType, computed } from "vue";
import {
  Database,
  DatabaseId,
  Environment,
  EnvironmentId,
  Project,
} from "../../types";

export type AlterType = "SINGLE_DB" | "MULTI_DB";

export type State = {
  alterType: AlterType;
  selectedDatabaseIdListForEnvironment: Map<EnvironmentId, DatabaseId[]>;
};

export default defineComponent({
  name: "ProjectStandardView",
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
    // MULTI_DB now supports selecting one database, which can be a replacement
    // of SINGLE_DB.
    // So SINGLE_DB is only needed and available for VCS workflow.
    // And we won't provide a radio button group for single/multi selection in
    // the future.
    watch(
      () => props.project?.workflowType,
      (type) => {
        if (type === "VCS") {
          props.state.alterType = "SINGLE_DB";
        } else {
          props.state.alterType = "MULTI_DB";
        }
      },
      {
        immediate: true,
      }
    );

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

    return {
      databaseListGroupByEnvironment,
      toggleDatabaseIdForEnvironment,
      isDatabaseSelectedForEnvironment,
      getAllSelectionStateForEnvironment,
      toggleAllDatabasesSelectionForEnvironment,
      selectDatabase,
    };
  },
});
</script>
