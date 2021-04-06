<template>
  <BBOutline
    :id="'database'"
    :title="'Databases'"
    :itemList="databaseListByEnvironment"
    :allowCollapse="false"
  />
</template>

<script lang="ts">
import { computed, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";

import {
  ALL_DATABASE_NAME,
  Database,
  Environment,
  EnvironmentId,
} from "../types";
import { databaseSlug, allowDatabaseAccess } from "../utils";
import { BBOutlineItem } from "../bbkit/types";

interface LocalState {
  databaseList: Database[];
}

export default {
  name: "DatabaseListSidePanel",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      databaseList: [],
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const environmentList = computed(() => {
      return cloneDeep(
        store.getters["environment/environmentList"]()
      ).reverse();
    });

    const prepareDatabaseList = () => {
      store
        .dispatch("database/fetchDatabaseListByUser", currentUser.value.id)
        .then((databaseList) => {
          state.databaseList = databaseList;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareDatabaseList);

    const databaseListByEnvironment = computed(() => {
      const envToDbMap: Map<EnvironmentId, BBOutlineItem[]> = new Map();
      for (const environment of environmentList.value) {
        envToDbMap.set(environment.id, []);
      }
      for (const database of state.databaseList) {
        if (database.name != ALL_DATABASE_NAME) {
          const dbList = envToDbMap.get(database.instance.environment.id)!;
          let databaseName = database.name;
          if (!allowDatabaseAccess(database, currentUser.value, "RW")) {
            databaseName += " (read)";
          }
          if (database.ownerId == currentUser.value.id) {
            databaseName += " (owner)";
          }
          dbList.push({
            id: database.id,
            name: databaseName,
            link: `/db/${databaseSlug(database)}`,
          });
        }
      }
      return environmentList.value.map(
        (environment: Environment): BBOutlineItem => {
          return {
            id: "env." + environment.id,
            name: environment.name,
            childList: envToDbMap.get(environment.id),
            childCollapse: true,
          };
        }
      );
    });

    return {
      environmentList,
      databaseListByEnvironment,
    };
  },
};
</script>
