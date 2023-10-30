<template>
  <BBOutline
    id="bb.recent-databases"
    :title="$t('database.recent')"
    :item-list="recentDatabaseItemList"
    :allow-collapse="false"
    outline-item-class="pt-0.5 pb-0.5"
  />
</template>

<script lang="ts" setup>
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useDebounce, useStorage } from "@vueuse/core";
import { cloneDeep, uniqBy } from "lodash-es";
import { computed, h, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBOutline, type BBOutlineItem } from "@/bbkit";
import {
  useEnvironmentV1List,
  useDatabaseV1Store,
  useCurrentUserV1,
  usePolicyV1Store,
  useDBGroupStore,
  useProjectV1ListByCurrentUser,
} from "@/store";
import {
  DEFAULT_PROJECT_V1_NAME,
  UNKNOWN_ID,
  UNKNOWN_USER_NAME,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import {
  databaseV1Slug,
  environmentV1Name,
  projectV1Slug,
  isDatabaseV1Accessible,
  uidFromSlug,
  keyBy,
  extractUserUID,
} from "@/utils";
import DatabaseGroupIcon from "./DatabaseGroupIcon.vue";
import EngineIcon from "./Icon/EngineIcon.vue";

const { t } = useI18n();
const databaseV1Store = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const rawEnvironmentList = useEnvironmentV1List();
const { projectList } = useProjectV1ListByCurrentUser();

type RecentVisitDatabase = {
  uid: string;
};

const useRecentDatabaseList = () => {
  const me = useCurrentUserV1();
  const STORAGE_KEY_PREFIX = "bb.ui.recent-database-list";
  const MAX_HISTORY = 10;
  // The format of storage key if {HARDCODED_PREFIX}.#{USER_UID}
  // to provide separate storage for each user in a same browser.
  const KEY = `${STORAGE_KEY_PREFIX}.#${extractUserUID(me.value.name)}`;

  const route = useRoute();
  const recentList = useStorage(KEY, [] as RecentVisitDatabase[]);
  const uid = computed(() => {
    if (route.matched.length === 0) {
      return undefined;
    }
    if (route.name === "workspace.database.detail") {
      const slug = route.params.databaseSlug;
      if (!slug || typeof slug !== "string") {
        return undefined;
      }
      const uid = uidFromSlug(slug);
      if (!uid || uid === String(UNKNOWN_ID)) {
        return undefined;
      }
      return uid;
    }
    return undefined;
  });

  watch(
    // Debounce the listener so we can skip internal immediate redirection
    useDebounce(uid, 50),
    (uid) => {
      if (!uid) return;
      if (uid === String(UNKNOWN_ID)) return;
      const list = [...recentList.value];
      const index = list.findIndex((item) => {
        return item.uid === uid;
      });
      if (index >= 0) {
        // current page exists in the history already
        // pull it out before next step
        list.splice(index, 1);
      }
      // then prepend the latest item to the queue
      list.unshift({ uid });

      // ensure the queue's length
      // should be no more than (MAX_HISTORY)
      while (list.length > MAX_HISTORY) {
        list.pop();
      }
      recentList.value = uniqBy(list, (item) => item.uid);
    },
    {
      immediate: true,
    }
  );

  return recentList;
};

// Reserve the environment list, put "Prod" to the top.
const environmentList = computed(() =>
  cloneDeep(rawEnvironmentList.value).reverse()
);

const recentDatabaseList = useRecentDatabaseList();

// Prepare policy list for checking if user has access to the database.
const preparePolicyList = () => {
  usePolicyV1Store().fetchPolicies({
    policyType: PolicyType.WORKSPACE_IAM,
    resourceType: PolicyResourceType.WORKSPACE,
  });
};

watchEffect(preparePolicyList);

// Prepare database and database group list.
const prepareDataList = () => {
  // It will also be called when user logout
  if (currentUserV1.value.name !== UNKNOWN_USER_NAME) {
    databaseV1Store.searchDatabaseList({
      parent: "instances/-",
    });
    dbGroupStore.fetchAllDatabaseGroupList();
  }
};

watchEffect(prepareDataList);

// Use this to make the list reactive when project is transferred.
const databaseList = computed(() => {
  return databaseV1Store
    .databaseListByUser(currentUserV1.value)
    .filter((db) => db.syncState === State.ACTIVE)
    .filter((database) =>
      isDatabaseV1Accessible(database, currentUserV1.value)
    );
});

const recentDatabaseItemList = computed(() => {
  const databaseMap = keyBy(databaseList.value, (db) => db.uid);
  const recentDatabaseItemList: BBOutlineItem[] = [];
  recentDatabaseList.value.forEach((item) => {
    const db = databaseMap.get(item.uid);
    if (db) {
      recentDatabaseItemList.push({
        id: `bb.database.${db.uid}`,
        name: `${db.databaseName} (${db.instanceEntity.title})`,
        link: `/db/${databaseV1Slug(db)}`,
        prefix: h(EngineIcon, {
          class: "shrink-0",
          engine: db.instanceEntity.engine,
        }),
      });
    }
  });
  return recentDatabaseItemList;
});

const databaseListByEnvironment = computed(() => {
  const envToDbMap: Map<string, BBOutlineItem[]> = new Map();
  for (const environment of environmentList.value) {
    envToDbMap.set(environment.name, []);
  }
  const list = [...databaseList.value].filter(
    (db) =>
      db.projectEntity.tenantMode !== TenantMode.TENANT_MODE_ENABLED &&
      db.project !== DEFAULT_PROJECT_V1_NAME
  );
  list.sort((a: any, b: any) => {
    return a.name.localeCompare(b.name);
  });
  for (const database of list) {
    const dbList = envToDbMap.get(String(database.effectiveEnvironment))!;
    // dbList may be undefined if the environment is archived
    if (dbList) {
      dbList.push({
        id: `bb.database.${database.uid}`,
        name: `${database.databaseName} (${database.instanceEntity.title})`,
        link: `/db/${databaseV1Slug(database)}`,
        prefix: h(EngineIcon, {
          engine: database.instanceEntity.engine,
        }),
      });
    }
  }

  return environmentList.value
    .filter((environment) => {
      const items = envToDbMap.get(environment.name) ?? [];
      return items.length > 0;
    })
    .map((environment): BBOutlineItem => {
      return {
        id: `bb.env.${environment.uid}`,
        name: environmentV1Name(environment),
        childList: envToDbMap.get(environment.name),
        childCollapse: true,
      };
    });
});

const tenantDatabaseListByProject = computed((): BBOutlineItem[] => {
  const dbList = databaseList.value.filter(
    (db) =>
      db.projectEntity.tenantMode === TenantMode.TENANT_MODE_ENABLED &&
      db.project !== DEFAULT_PROJECT_V1_NAME
  );
  const sortedProjectList = projectList.value
    .map((p) => p)
    .sort((a, b) => a.name.localeCompare(b.name));
  const databaseListByProject = sortedProjectList.map((project) => {
    const databaseList = dbList.filter((db) => db.project === project.name);
    return {
      project,
      databaseList,
    };
  });
  const databaseGroupListByProject = sortedProjectList.map((project) => {
    const databaseGroupList = dbGroupStore.getDBGroupListByProjectName(
      project.name
    );
    return {
      project,
      databaseGroupList,
    };
  });

  const outlineItemList: BBOutlineItem[] = sortedProjectList.map((project) => {
    const databaseList =
      databaseListByProject.find((item) => item.project === project)
        ?.databaseList || [];
    const databaseGroupList =
      databaseGroupListByProject.find((item) => item.project === project)
        ?.databaseGroupList || [];
    return {
      id: `bb.project.${project.uid}`,
      name: project.title,
      childList: [
        ...databaseList.map<BBOutlineItem>((db) => ({
          id: `bb.project.${project.uid}.database.${db.databaseName}`,
          name: `${db.databaseName} (${db.instanceEntity.title})`,
          link: `/project/${projectV1Slug(project)}#databases`,
          prefix: h(EngineIcon, {
            engine: db.instanceEntity.engine,
          }),
        })),
        ...databaseGroupList.map<BBOutlineItem>((dbGroup) => ({
          id: `bb.project.${project.uid}.databaseGroup.${dbGroup.name}`,
          name: dbGroup.databaseGroupName,
          link: `/project/${projectV1Slug(project)}/database-groups/${
            dbGroup.databaseGroupName
          }`,
          prefix: h(DatabaseGroupIcon, {
            class: "w-4 h-auto",
          }),
        })),
      ],
      childCollapse: true,
    } as BBOutlineItem;
  });

  return outlineItemList.filter(
    (item) => item.childList && item.childList.length > 0
  );
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
</script>
