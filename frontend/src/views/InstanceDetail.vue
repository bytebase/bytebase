<template>
  <div class="p-6 space-y-2">
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
        <InstanceForm :instance="instance" />
      </NTabPane>
      <NTabPane name="DATABASES" :tab="$t('common.databases')">
        <DatabaseV1Table
          mode="INSTANCE"
          table-class="border"
          :scroll-on-page-change="false"
          :database-list="databaseV1List"
        />
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
      :environment-id="environment?.uid"
      :instance-id="instance.uid"
      @dismiss="state.showCreateDatabaseModal = false"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton, NTabPane, NTabs } from "naive-ui";
import { ClientError } from "nice-grpc-web";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import { EngineIcon } from "@/components/Icon";
import InstanceForm from "@/components/InstanceForm/";
import { InstanceRoleTable, DatabaseV1Table, Drawer } from "@/components/v2";
import {
  pushNotification,
  useDBSchemaV1Store,
  useCurrentUserV1,
  useInstanceV1Store,
  useEnvironmentV1Store,
  useDatabaseV1Store,
} from "@/store";
import { State } from "@/types/proto/v1/common";
import {
  idFromSlug,
  instanceV1HasCreateDatabase,
  instanceV1Name,
  hasWorkspacePermissionV2,
  hasWorkspaceLevelProjectPermission,
} from "@/utils";

interface LocalState {
  showCreateDatabaseModal: boolean;
  syncingSchema: boolean;
}

const props = defineProps({
  instanceSlug: {
    required: true,
    type: String,
  },
});

const instanceV1Store = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const { t } = useI18n();

const currentUser = useCurrentUserV1();

const state = reactive<LocalState>({
  showCreateDatabaseModal: false,
  syncingSchema: false,
});

const instanceId = computed(() => {
  return idFromSlug(props.instanceSlug);
});
const instance = computed(() => {
  return instanceV1Store.getInstanceByUID(String(instanceId.value));
});
const environment = computed(() => {
  return useEnvironmentV1Store().getEnvironmentByName(
    instance.value.environment
  );
});

watchEffect(() => {
  databaseStore.fetchDatabaseList({
    parent: instance.value.name,
  });
});

const databaseV1List = computed(() => {
  return databaseStore.databaseListByInstance(instance.value.name);
});

const instanceRoleList = computed(() => {
  return instanceV1Store.getInstanceRoleListByName(instance.value.name);
});

const allowSyncInstance = computed(() => {
  return (
    instance.value.state === State.ACTIVE &&
    hasWorkspacePermissionV2(currentUser.value, "bb.instances.sync")
  );
});

const allowCreateDatabase = computed(() => {
  return (
    instance.value.state === State.ACTIVE &&
    hasWorkspaceLevelProjectPermission(currentUser.value, "bb.issues.create") &&
    instanceV1HasCreateDatabase(instance.value)
  );
});

const syncSchema = async () => {
  state.syncingSchema = true;
  try {
    await instanceV1Store.syncInstance(instance.value).then(() => {
      return databaseStore.fetchDatabaseList({
        parent: instance.value.name,
      });
    });
    // Clear the db schema metadata cache entities.
    // So we will re-fetch new values when needed.
    const dbSchemaStore = useDBSchemaV1Store();
    const databaseList = useDatabaseV1Store().databaseListByInstance(
      instance.value.name
    );
    databaseList.forEach((database) =>
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
</script>
