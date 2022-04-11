<template>
  <BBOutline
    id="database"
    :title="$t('common.databases')"
    :item-list="mixedDatabaseList"
    :allow-collapse="false"
  />
</template>

<script lang="ts">
import { computed, defineComponent, watchEffect } from "vue";
import { useStore } from "vuex";
import { cloneDeep, groupBy, uniqBy } from "lodash-es";
import { Database, Environment, EnvironmentId, UNKNOWN_ID } from "../types";
import {
  databaseSlug,
  environmentName,
  parseDatabaseNameByTemplate,
  projectSlug,
} from "../utils";
import { BBOutlineItem } from "../bbkit/types";
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useEnvironmentList, useCurrentUser, useLabelStore } from "@/store";
import { storeToRefs } from "pinia";

export default defineComponent({
  name: "DatabaseListSidePanel",
  setup() {
    const { t } = useI18n();
    const store = useStore();
    const labelStore = useLabelStore();
    const router = useRouter();

    const currentUser = useCurrentUser();

    const rawEnvironmentList = useEnvironmentList();
    const environmentList = computed(() =>
      cloneDeep(rawEnvironmentList.value).reverse()
    );

    const prepareList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store.dispatch("database/fetchDatabaseList");
      }

      labelStore.fetchLabelList();
    };

    watchEffect(prepareList);

    // Use this to make the list reactive when project is transferred.
    const databaseList = computed((): Database[] => {
      return store.getters["database/databaseListByPrincipalId"](
        currentUser.value.id
      );
    });

    // Use this to parse database name from name template
    const { labelList } = storeToRefs(labelStore);

    const databaseListByEnvironment = computed(() => {
      const envToDbMap: Map<EnvironmentId, BBOutlineItem[]> = new Map();
      for (const environment of environmentList.value) {
        envToDbMap.set(environment.id, []);
      }
      const list = [...databaseList.value].filter(
        (db) => db.project.tenantMode !== "TENANT"
      );
      list.sort((a: any, b: any) => {
        return a.name.localeCompare(b.name);
      });
      for (const database of list) {
        const dbList = envToDbMap.get(database.instance.environment.id)!;
        // dbList may be undefined if the environment is archived
        if (dbList) {
          dbList.push({
            id: `bb.database.${database.id}`,
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
            id: `bb.env.${environment.id}`,
            name: environmentName(environment),
            childList: envToDbMap.get(environment.id),
            childCollapse: true,
          };
        });
    });

    const tenantDatabaseListByProject = computed((): BBOutlineItem[] => {
      if (labelList.value.length === 0) {
        // wait for the labelList to be loaded
        // to prevent UI jitter
        return [];
      }

      const list = databaseList.value.filter(
        (db) => db.project.tenantMode === "TENANT"
      );
      // In case that each `db.project` is not reference equal
      // we run a uniq() on the list by project.id
      const projectList = uniqBy(
        list.map((db) => db.project),
        (project) => project.id
      );
      // Sort the list as what <ProjectListSidePanel /> does
      projectList.sort((a, b) => a.name.localeCompare(b.name));
      // Then group databaseList by project
      const databaseListGroupByProject = projectList.map((project) => {
        const databaseList = list.filter((db) => db.project.id === project.id);
        return {
          project,
          databaseList,
        };
      });
      // Map groups to `BBOutlineItem[]`
      const itemList = databaseListGroupByProject.map(
        ({ project, databaseList }) => {
          const databaseListGroupByName = groupBy(databaseList, (db) => {
            if (project.dbNameTemplate) {
              // parse db name from template if possible
              return parseDatabaseNameByTemplate(
                db.name,
                project.dbNameTemplate,
                labelList.value
              );
            } else {
              // use raw db.name otherwise
              return db.name;
            }
          });
          const databaseListGroupByNameAndCount = Object.keys(
            databaseListGroupByName
          ).map((name) => {
            return {
              name,
              count: databaseListGroupByName[name].length,
            };
          });
          return {
            id: `bb.project.${project.id}.databases`,
            name: project.name,
            childList: databaseListGroupByNameAndCount.map(
              ({ name, count }) => {
                return {
                  id: `bb.project.${project.id}.database.${name}`,
                  name: `${name} (${count})`,
                  link: `/project/${projectSlug(project)}`,
                } as BBOutlineItem;
              }
            ),
            childCollapse: true,
          } as BBOutlineItem;
        }
      );
      return itemList;
    });

    const mixedDatabaseList = computed(() => {
      return [
        ...databaseListByEnvironment.value,
        ...tenantDatabaseListByProject.value,
      ];
    });

    const kbarActions = computed((): Action[] => {
      const actions = mixedDatabaseList.value.flatMap((group: BBOutlineItem) =>
        group.childList!.map((item) =>
          defineAction({
            // `item.id` is namespaced already
            // so here `id` looks like
            // "bb.database.7001" for non-tenant databases
            // "bb.project.3007.database.db3" for tenant databases
            id: item.id,
            section: t("common.databases"),
            name: item.name,
            // `group.name` is also a keyword to provide better search
            // e.g. "blog" under "staged" now can be searched by "bl st"
            // also "blog" under "HR system" (a project) can be searched by "bl hr"
            keywords: `database db ${group.name}`,
            data: {
              tags: [group.name],
            },
            perform: () => {
              router.push(item.link!);
            },
          })
        )
      );
      return actions;
    });
    useRegisterActions(kbarActions);

    return {
      mixedDatabaseList,
    };
  },
});
</script>
