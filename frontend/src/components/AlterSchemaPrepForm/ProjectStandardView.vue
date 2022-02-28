<template>
  <!-- eslint-disable vue/no-mutating-props -->

  <div v-if="project?.workflowType === 'UI'" class="my-2 textlabel -ml-1">
    <div class="radio-set-row">
      <div class="radio">
        <label class="label">
          <input
            v-model="state.alterType"
            tabindex="-1"
            type="radio"
            class="btn"
            value="SINGLE_DB"
          />
          {{ $t("alter-schema.alter-single-db") }}
        </label>
      </div>
      <div class="radio">
        <label class="label">
          <input
            v-model="state.alterType"
            tabindex="-1"
            type="radio"
            class="btn"
            value="MULTI_DB"
          />
          {{ $t("alter-schema.alter-multiple-db") }}
        </label>
      </div>
    </div>
  </div>

  <template v-if="state.alterType === 'MULTI_DB'">
    <!-- multiple stage view -->
    <div class="textinfolabel">
      {{ $t("alter-schema.alter-multiple-db-info") }}
    </div>
    <div class="space-y-4">
      <div v-for="(environment, envIndex) in environmentList" :key="envIndex">
        <div class="mb-2 mt-4">{{ environment.name }}</div>
        <div class="relative bg-white rounded-md -space-y-px">
          <template
            v-for="(database, dbIndex) in databaseList.filter(
              (item) => item.instance.environment.id == environment.id
            )"
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
                  v-if="database.syncStatus == 'OK'"
                  type="radio"
                  class="btn"
                  :checked="
                    state.selectedDatabaseIdForEnvironment.get(
                      environment.id
                    ) == database.id
                  "
                  @change="
                    selectDatabaseIdForEnvironment(database.id, environment.id)
                  "
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
          <label
            class="border-control-border relative border p-3 flex flex-col cursor-pointer md:pl-4 md:pr-6 md:grid md:grid-cols-3"
          >
            <div class="radio space-x-2 text-sm">
              <input
                type="radio"
                class="btn"
                :checked="
                  state.selectedDatabaseIdForEnvironment.get(environment.id)
                    ? false
                    : true
                "
                @input="clearDatabaseIdForEnvironment(environment.id)"
              />
              <span class="ml-3 font-medium text-main uppercase">{{
                $t("common.skip")
              }}</span>
            </div>
          </label>
        </div>
      </div>
    </div>
  </template>
  <template v-else>
    <!-- single stage view -->
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
import { defineComponent, PropType } from "vue";
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
  selectedDatabaseIdForEnvironment: Map<EnvironmentId, DatabaseId>;
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
    const selectDatabaseIdForEnvironment = (
      databaseId: DatabaseId,
      environmentId: EnvironmentId
    ) => {
      props.state.selectedDatabaseIdForEnvironment.set(
        environmentId,
        databaseId
      );
    };

    const clearDatabaseIdForEnvironment = (environmentId: EnvironmentId) => {
      props.state.selectedDatabaseIdForEnvironment.delete(environmentId);
    };

    const selectDatabase = (db: Database) => {
      emit("select-database", db);
    };

    return {
      selectDatabaseIdForEnvironment,
      clearDatabaseIdForEnvironment,
      selectDatabase,
    };
  },
});
</script>
