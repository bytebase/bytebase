<template>
  <BBOutline
    :id="'database'"
    :title="$t('common.databases')"
    :item-list="databaseListByEnvironment"
    :allow-collapse="false"
  />
</template>

<script lang="ts">
import { computed, watchEffect } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";

import { Database, Environment, EnvironmentId, UNKNOWN_ID } from "../types";
import { databaseSlug, environmentName } from "../utils";
import { BBOutlineItem } from "../bbkit/types";
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useRouter } from "vue-router";

export default {
  name: "DatabaseListSidePanel",
  setup() {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const environmentList = computed(() => {
      return cloneDeep(
        store.getters["environment/environmentList"]()
      ).reverse();
    });

    const prepareDatabaseList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store.dispatch("database/fetchDatabaseList");
      }
    };

    watchEffect(prepareDatabaseList);

    // Use this to make the list reactive when project is transferred.
    const databaseList = computed((): Database[] => {
      return store.getters["database/databaseListByPrincipalId"](
        currentUser.value.id
      );
    });

    const databaseListByEnvironment = computed(() => {
      const envToDbMap: Map<EnvironmentId, BBOutlineItem[]> = new Map();
      for (const environment of environmentList.value) {
        envToDbMap.set(environment.id, []);
      }
      const list = [...databaseList.value];
      list.sort((a: any, b: any) => {
        return a.name.localeCompare(b.name);
      });
      for (const database of list) {
        const dbList = envToDbMap.get(database.instance.environment.id)!;
        // dbList may be undefined if the environment is archived
        if (dbList) {
          dbList.push({
            id: database.id.toString(),
            name: database.name,
            link: `/db/${databaseSlug(database)}`,
          });
        }
      }

      return environmentList.value
        .filter((environment: Environment) => {
          return envToDbMap.get(environment.id)!.length > 0;
        })
        .map((environment: Environment): BBOutlineItem => {
          return {
            id: "env." + environment.id,
            name: environmentName(environment),
            childList: envToDbMap.get(environment.id),
            childCollapse: true,
          };
        });
    });

    const kbarActions = computed((): Action[] => {
      const actions = databaseListByEnvironment.value.flatMap((env: any) =>
        env.childList.map((db: any) =>
          defineAction({
            // `db.id` is global unique, so need not to specify `env.id`
            // so here `id` looks like "bb.database.1234"
            id: `bb.database.${db.id}`,
            section: "Databases",
            name: db.name,
            keywords: "database db",
            data: {
              tags: [env.name],
            },
            perform: () => {
              router.push({ path: db.link });
            },
          })
        )
      );
      return actions;
    });
    useRegisterActions(kbarActions);

    return {
      environmentList,
      databaseList,
      databaseListByEnvironment,
    };
  },
};
</script>
