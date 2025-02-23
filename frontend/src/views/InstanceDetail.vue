<template>
  <div class="space-y-2 px-6" v-bind="$attrs">
    <ArchiveBanner v-if="instance.state === State.DELETED" />

    <div v-if="!embedded" class="flex items-center justify-between">
      <div class="flex items-center gap-x-2">
        <EngineIcon :engine="instance.engine" custom-class="!h-6" />
        <span class="text-lg font-medium">{{ instanceV1Name(instance) }}</span>
      </div>
    </div>

    <NTabs v-model:value="state.selectedTab">
      <template #suffix>
        <div class="flex items-center space-x-2">
          <InstanceSyncButton
            v-if="instance.state === State.ACTIVE"
            @sync-schema="syncSchema"
          />
          <NButton
            v-if="allowCreateDatabase"
            type="primary"
            @click.prevent="createDatabase"
          >
            <template #icon>
              <PlusIcon class="h-4 w-4" />
            </template>
            {{ $t("instance.new-database") }}
          </NButton>
        </div>
      </template>
      <NTabPane name="overview" :tab="$t('common.overview')">
        <InstanceForm class="-mt-2" :instance="instance">
          <InstanceFormBody :hide-archive-restore="hideArchiveRestore" />
          <InstanceFormButtons class="sticky bottom-0 z-10" />
        </InstanceForm>
      </NTabPane>
      <NTabPane name="databases" :tab="$t('common.databases')">
        <div class="space-y-2">
          <div
            class="w-full flex flex-col sm:flex-row items-start sm:items-end justify-between gap-2"
          >
            <AdvancedSearch
              v-model:params="state.params"
              class="flex-1"
              :autofocus="false"
              :placeholder="$t('database.filter-database')"
              :scope-options="scopeOptions"
              :readonly-scopes="readonlyScopes"
            />
            <DatabaseLabelFilter
              v-model:selected="state.selectedLabels"
              :database-list="databaseList"
              :placement="'left-start'"
            />
          </div>
          <DatabaseOperations :databases="selectedDatabases" />
          <DatabaseV1Table
            :key="`database-table.${instanceId}`"
            mode="INSTANCE"
            :show-selection="true"
            :database-list="filteredDatabaseList"
            :custom-click="true"
            :keyword="state.params.query.trim().toLowerCase()"
            @row-click="handleDatabaseClick"
            @update:selected-databases="handleDatabasesSelectionChanged"
          />
        </div>
      </NTabPane>
      <NTabPane name="users" :tab="$t('instance.users')">
        <InstanceRoleTable :instance-role-list="instanceRoleList" />
      </NTabPane>
    </NTabs>
  </div>

  <Drawer
    v-model:show="state.showCreateDatabaseModal"
    :title="$t('quick-action.create-db')"
  >
    <CreateDatabasePrepPanel
      :environment-name="environment?.name"
      :instance-name="instance.name"
      @dismiss="state.showCreateDatabaseModal = false"
    />
  </Drawer>
</template>

<script lang="tsx" setup>
import { useTitle } from "@vueuse/core";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NTabPane, NTabs } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter, useRoute } from "vue-router";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import { EngineIcon } from "@/components/Icon";
import InstanceSyncButton from "@/components/Instance/InstanceSyncButton.vue";
import {
  InstanceForm,
  Form as InstanceFormBody,
  Buttons as InstanceFormButtons,
} from "@/components/InstanceForm/";
import { InstanceRoleTable, Drawer } from "@/components/v2";
import DatabaseV1Table, {
  DatabaseOperations,
  DatabaseLabelFilter,
} from "@/components/v2/Model/DatabaseV1Table";
import { useBodyLayoutContext } from "@/layouts/common";
import {
  pushNotification,
  useDBSchemaV1Store,
  useInstanceV1Store,
  useEnvironmentV1Store,
  useAppFeature,
} from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import { UNKNOWN_ID, type ComposedDatabase } from "@/types";
import { State } from "@/types/proto/v1/common";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import {
  instanceV1HasCreateDatabase,
  instanceV1Name,
  hasWorkspaceLevelProjectPermissionInAnyProject,
  wrapRefAsPromise,
  autoDatabaseRoute,
  CommonFilterScopeIdList,
  filterDatabaseV1ByKeyword,
  extractEnvironmentResourceName,
  extractProjectResourceName,
} from "@/utils";
import type { SearchParams, SearchScope } from "@/utils";

const instanceHashList = ["overview", "databases", "users"] as const;
export type InstanceHash = (typeof instanceHashList)[number];
const isInstanceHash = (x: any): x is InstanceHash =>
  instanceHashList.includes(x);

interface LocalState {
  showCreateDatabaseModal: boolean;
  syncingSchema: boolean;
  selectedDatabaseNameList: Set<string>;
  selectedLabels: { key: string; value: string }[];
  params: SearchParams;
  selectedTab: InstanceHash;
}

const props = defineProps<{
  instanceId: string;
  embedded?: boolean;
  hideArchiveRestore?: boolean;
}>();

defineOptions({
  inheritAttrs: false,
});

if (!props.embedded) {
  const { overrideMainContainerClass } = useBodyLayoutContext();
  overrideMainContainerClass("!pb-0");
}

const { t } = useI18n();
const router = useRouter();
const instanceV1Store = useInstanceV1Store();
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");

const readonlyScopes = computed((): SearchScope[] => [
  { id: "instance", value: props.instanceId },
]);

const state = reactive<LocalState>({
  showCreateDatabaseModal: false,
  syncingSchema: false,
  selectedDatabaseNameList: new Set(),
  selectedLabels: [],
  params: {
    query: "",
    scopes: [...readonlyScopes.value],
  },
  selectedTab: "overview",
});

const route = useRoute();

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  [...CommonFilterScopeIdList, "project"]
);

watch(
  () => route.hash,
  (hash) => {
    const targetHash = hash.replace(/^#?/g, "") as InstanceHash;
    if (isInstanceHash(targetHash)) {
      state.selectedTab = targetHash;
    }
  },
  { immediate: true }
);

watch(
  () => state.selectedTab,
  (tab) => {
    router.replace({
      hash: `#${tab}`,
      query: route.query,
    });
  },
  { immediate: true }
);

const instance = computed(() => {
  return instanceV1Store.getInstanceByName(
    `${instanceNamePrefix}${props.instanceId}`
  );
});

const environment = computed(() => {
  return useEnvironmentV1Store().getEnvironmentByName(
    instance.value.environment
  );
});

const { databaseList, listCache, ready } = useDatabaseV1List(
  instance.value.name
);

const selectedEnvironment = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "environment")?.value ??
    `${UNKNOWN_ID}`
  );
});

const selectedProject = computed(() => {
  return state.params.scopes.find((scope) => scope.id === "project")?.value;
});

const filteredDatabaseList = computed(() => {
  let list = databaseList.value;
  if (selectedEnvironment.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (db) =>
        extractEnvironmentResourceName(db.effectiveEnvironment) ===
        selectedEnvironment.value
    );
  }
  if (selectedProject.value) {
    list = list.filter(
      (db) => extractProjectResourceName(db.project) === selectedProject.value
    );
  }
  const keyword = state.params.query.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
        "project",
      ])
    );
  }
  const labels = state.selectedLabels;
  if (labels.length > 0) {
    list = list.filter((db) => {
      return labels.some((kv) => db.labels[kv.key] === kv.value);
    });
  }
  return list;
});

const instanceRoleList = computed(() => {
  return instance.value.roles;
});

const allowCreateDatabase = computed(() => {
  return (
    databaseChangeMode.value === DatabaseChangeMode.PIPELINE &&
    instance.value.state === State.ACTIVE &&
    hasWorkspaceLevelProjectPermissionInAnyProject("bb.issues.create") &&
    instanceV1HasCreateDatabase(instance.value)
  );
});

const syncSchema = async (enableFullSync: boolean) => {
  await instanceV1Store.syncInstance(instance.value.name, enableFullSync);
  // Remove the database list cache for the instance.
  listCache.deleteCache(instance.value.name);
  await wrapRefAsPromise(ready, true);
  if (enableFullSync) {
    // Clear the db schema metadata cache entities.
    // So we will re-fetch new values when needed.
    const dbSchemaStore = useDBSchemaV1Store();
    databaseList.value.forEach((database) =>
      dbSchemaStore.removeCache(database.name)
    );
  }
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t(
      "instance.successfully-synced-schema-for-instance-instance-value-name",
      [instance.value.title]
    ),
  });
};

const createDatabase = () => {
  state.showCreateDatabaseModal = true;
};

useTitle(instance.value.title);

const handleDatabaseClick = (event: MouseEvent, database: ComposedDatabase) => {
  const url = router.resolve(autoDatabaseRoute(router, database)).fullPath;
  if (event.ctrlKey || event.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};

const handleDatabasesSelectionChanged = (
  selectedDatabaseNameList: Set<string>
): void => {
  state.selectedDatabaseNameList = selectedDatabaseNameList;
};

const selectedDatabases = computed((): ComposedDatabase[] => {
  return databaseList.value.filter((db) =>
    state.selectedDatabaseNameList.has(db.name)
  );
});
</script>
