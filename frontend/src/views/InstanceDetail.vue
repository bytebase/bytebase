<template>
  <div class="px-6 space-y-2 mb-4">
    <ArchiveBanner v-if="instance.state === State.DELETED" />

    <div class="flex items-center justify-between">
      <div class="flex items-center gap-x-2">
        <EngineIcon :engine="instance.engine" custom-class="!h-6" />
        <span class="text-lg font-medium">{{ instanceV1Name(instance) }}</span>
      </div>
      <div class="flex items-center gap-x-2">
        <NButton
          v-if="allowSyncInstance"
          :loading="state.syncingSchema"
          @click.prevent="syncSchema"
        >
          <template v-if="state.syncingSchema">
            {{ $t("instance.syncing") }}
          </template>
          <template v-else>
            {{ $t("common.sync-now") }}
          </template>
        </NButton>
        <NButton
          v-if="allowCreateDatabase"
          type="primary"
          @click.prevent="createDatabase"
        >
          {{ $t("instance.new-database") }}
        </NButton>
      </div>
    </div>

    <NTabs>
      <NTabPane name="OVERVIEW" :tab="$t('common.overview')">
        <InstanceForm class="-mt-2" :instance="instance">
          <InstanceFormBody />
          <InstanceFormButtons class="sticky bottom-0 z-10" />
        </InstanceForm>
      </NTabPane>
      <NTabPane name="DATABASES" :tab="$t('common.databases')">
        <div class="space-y-2">
          <DatabaseOperations :databases="selectedDatabases" />
          <DatabaseV1Table
            :key="`database-table.${instanceId}`"
            mode="INSTANCE"
            :show-selection="true"
            :database-list="databaseList"
            :custom-click="true"
            @row-click="handleDatabaseClick"
            @update:selected-databases="handleDatabasesSelectionChanged"
          />
        </div>
      </NTabPane>
      <NTabPane name="USERS" :tab="$t('instance.users')">
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

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { NButton, NTabPane, NTabs } from "naive-ui";
import type { ClientError } from "nice-grpc-web";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import { EngineIcon } from "@/components/Icon";
import {
  InstanceForm,
  Form as InstanceFormBody,
  Buttons as InstanceFormButtons,
} from "@/components/InstanceForm/";
import { InstanceRoleTable, Drawer } from "@/components/v2";
import DatabaseV1Table, {
  DatabaseOperations,
} from "@/components/v2/Model/DatabaseV1Table";
import { useBodyLayoutContext } from "@/layouts/common";
import {
  pushNotification,
  useDBSchemaV1Store,
  useInstanceV1Store,
  useEnvironmentV1Store,
} from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type { ComposedDatabase } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  instanceV1HasCreateDatabase,
  instanceV1Name,
  hasWorkspacePermissionV2,
  hasWorkspaceLevelProjectPermissionInAnyProject,
  databaseV1Url,
  wrapRefAsPromise,
} from "@/utils";

interface LocalState {
  showCreateDatabaseModal: boolean;
  syncingSchema: boolean;
  selectedDatabaseNameList: Set<string>;
}

const props = defineProps<{
  instanceId: string;
}>();

const { overrideMainContainerClass } = useBodyLayoutContext();
overrideMainContainerClass("!pb-0");

const { t } = useI18n();
const router = useRouter();
const instanceV1Store = useInstanceV1Store();

const state = reactive<LocalState>({
  showCreateDatabaseModal: false,
  syncingSchema: false,
  selectedDatabaseNameList: new Set(),
});

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

const instanceRoleList = computed(() => {
  return instance.value.roles;
});

const allowSyncInstance = computed(() => {
  return (
    instance.value.state === State.ACTIVE &&
    hasWorkspacePermissionV2("bb.instances.sync")
  );
});

const allowCreateDatabase = computed(() => {
  return (
    instance.value.state === State.ACTIVE &&
    hasWorkspaceLevelProjectPermissionInAnyProject("bb.issues.create") &&
    instanceV1HasCreateDatabase(instance.value)
  );
});

const syncSchema = async () => {
  state.syncingSchema = true;
  try {
    await instanceV1Store.syncInstance(instance.value);
    // Remove the database list cache for the instance.
    listCache.cacheMap.delete(listCache.getCacheKey(instance.value.name));
    await wrapRefAsPromise(ready, true);
    // Clear the db schema metadata cache entities.
    // So we will re-fetch new values when needed.
    const dbSchemaStore = useDBSchemaV1Store();
    databaseList.value.forEach((database) =>
      dbSchemaStore.removeCache(database.name)
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t(
        "instance.successfully-synced-schema-for-instance-instance-value-name",
        [instance.value.title]
      ),
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t(
        "instance.failed-to-sync-schema-for-instance-instance-value-name",
        [instance.value.title]
      ),
      description: (error as ClientError).details,
    });
  } finally {
    state.syncingSchema = false;
  }
};

const createDatabase = () => {
  state.showCreateDatabaseModal = true;
};

useTitle(instance.value.title);

const handleDatabaseClick = (event: MouseEvent, database: ComposedDatabase) => {
  const url = databaseV1Url(database);
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
