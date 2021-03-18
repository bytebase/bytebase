<template>
  <BBOutline
    :id="'database'"
    :title="'Databases'"
    :itemList="databaseListByEnvironment"
    :allowCollapse="false"
  />
</template>

<script lang="ts">
import { computed, watchEffect, ComputedRef } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";

import {
  ALL_DATABASE_NAME,
  Database,
  Environment,
  EnvironmentId,
} from "../types";
import { instanceSlug, databaseSlug } from "../utils";
import { BBOutlineItem } from "../bbkit/types";

export default {
  name: "DatabaseListSidePanel",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const environmentList = computed(() => {
      return cloneDeep(
        store.getters["environment/environmentList"]()
      ).reverse();
    });

    const prepareDatabaseList = () => {
      store
        .dispatch("database/fetchDatabaseListByUser", currentUser.value.id)
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareDatabaseList);

    const databaseListByEnvironment = computed(() => {
      const databaseList = store.getters["database/databaseListByUserId"](
        currentUser.value.id
      );
      const envToDbMap: Map<EnvironmentId, BBOutlineItem[]> = new Map();
      for (const environment of environmentList.value) {
        envToDbMap.set(environment.id, []);
      }
      for (const database of databaseList) {
        if (database.name != ALL_DATABASE_NAME) {
          const dbList = envToDbMap.get(database.instance.environment.id)!;
          dbList.push({
            id: database.id,
            name: database.name,
            link: `/instance/${instanceSlug(
              database.instance
            )}/db/${databaseSlug(database)}`,
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
