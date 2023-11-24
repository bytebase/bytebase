import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { computed, unref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useEnvironmentV1List,
} from "@/store";
import {
  ComposedDatabase,
  ComposedProject,
  DEFAULT_PROJECT_V1_NAME,
  MaybeRef,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  isDatabaseV1Accessible,
  groupBy,
  environmentV1Name,
  databaseV1Slug,
  sortDatabaseV1List,
} from "@/utils";

export const useDatabaseActions = (
  databaseList: MaybeRef<ComposedDatabase[]>
) => {
  const { t } = useI18n();
  const router = useRouter();
  const me = useCurrentUserV1();
  const environmentList = useEnvironmentV1List();

  const accessibleDatabaseList = computed(() => {
    return unref(databaseList)
      .filter((db) => db.syncState === State.ACTIVE)
      .filter((db) => isDatabaseV1Accessible(db, me.value));
  });

  const sortedDatabaseList = computed(() => {
    return sortDatabaseV1List(accessibleDatabaseList.value);
  });

  const databaseListByEnvironment = computed(() => {
    const databasesByEnv = groupBy(
      sortedDatabaseList.value,
      (db) => db.effectiveEnvironment
    );

    return environmentList.value
      .filter((environment) => {
        const group = databasesByEnv.get(environment.name) ?? [];
        return group.length > 0;
      })
      .map((environment) => {
        const databases = databasesByEnv.get(environment.name)!;
        return {
          id: `bb.env.${environment.uid}`,
          name: environmentV1Name(environment),
          children: databases.map((db) => ({
            id: `bb.database.${db.uid}`,
            name: `${db.databaseName} (${db.instanceEntity.title})`,
            link: `/db/${databaseV1Slug(db)}`,
          })),
        };
      });
  });

  const kbarActions = computed((): Action[] => {
    const actions = databaseListByEnvironment.value.flatMap((group) =>
      group.children!.map((item) =>
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
};

export const useProjectDatabaseActions = (
  project: MaybeRef<ComposedProject>
) => {
  const projectDatabaseList = computed(() => {
    return useDatabaseV1Store().databaseListByProject(unref(project).name);
  });
  useDatabaseActions(projectDatabaseList);
};

export const useGlobalDatabaseActions = () => {
  const me = useCurrentUserV1();
  // Use this to make the list reactive when project is transferred.
  const databaseList = computed(() => {
    return useDatabaseV1Store()
      .databaseListByUser(me.value)
      .filter((db) => db.project !== DEFAULT_PROJECT_V1_NAME);
  });
  useDatabaseActions(databaseList);
};
