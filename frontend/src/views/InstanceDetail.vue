<template>
  <div class="py-4 space-y-4">
    <ArchiveBanner v-if="instance.state === State.DELETED" />

    <div class="px-6 space-y-6">
      <InstanceForm :instance="instance" />
      <NTabs>
        <template #suffix>
          <div class="flex items-center gap-x-4">
            <NButton
              v-if="allowEdit"
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
              v-if="
                instance.state === State.ACTIVE &&
                instanceV1HasCreateDatabase(instance)
              "
              type="primary"
              @click.prevent="createDatabase"
            >
              {{ $t("instance.new-database") }}
            </NButton>
          </div>
        </template>

        <NTabPane name="DATABASES" :tab="$t('common.databases')">
          <DatabaseV1Table
            mode="INSTANCE"
            :scroll-on-page-change="false"
            :database-list="databaseV1List"
          />
        </NTabPane>
        <NTabPane name="USERS" :tab="$t('instance.users')">
          <InstanceRoleTable :instance-role-list="instanceRoleList" />
        </NTabPane>
      </NTabs>
      <template v-if="allowArchiveOrRestore">
        <template v-if="instance.state === State.ACTIVE">
          <BBButtonConfirm
            :style="'ARCHIVE'"
            :button-text="$t('instance.archive-this-instance')"
            :ok-text="$t('common.archive')"
            :require-confirm="true"
            :confirm-title="
              $t('instance.archive-instance-instance-name', [instance.title])
            "
            :confirm-description="
              $t(
                'instance.archived-instances-will-not-be-shown-on-the-normal-interface-you-can-still-restore-later-from-the-archive-page'
              )
            "
            @confirm="doArchive"
          />
        </template>
        <template v-else-if="instance.state === State.DELETED">
          <BBButtonConfirm
            :style="'RESTORE'"
            :button-text="$t('instance.restore-this-instance')"
            :ok-text="$t('instance.restore')"
            :require-confirm="true"
            :confirm-title="
              $t('instance.restore-instance-instance-name-to-normal-state', [
                instance.title,
              ])
            "
            :confirm-description="''"
            @confirm="doRestore"
          />
        </template>
      </template>
    </div>
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

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.instance-count"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { NButton, NTabPane, NTabs } from "naive-ui";
import { useI18n } from "vue-i18n";
import { ClientError } from "nice-grpc-web";

import {
  idFromSlug,
  hasWorkspacePermissionV1,
  instanceV1HasCreateDatabase,
  isMemberOfProjectV1,
} from "@/utils";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import InstanceForm from "@/components/InstanceForm/";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import { InstanceRoleTable, DatabaseV1Table, Drawer } from "@/components/v2";
import {
  pushNotification,
  useSubscriptionV1Store,
  useDBSchemaV1Store,
  useCurrentUserV1,
  useInstanceV1Store,
  useEnvironmentV1Store,
  useGracefulRequest,
  useDatabaseV1Store,
} from "@/store";
import { State } from "@/types/proto/v1/common";

interface LocalState {
  showCreateDatabaseModal: boolean;
  syncingSchema: boolean;
  showFeatureModal: boolean;
}

const props = defineProps({
  instanceSlug: {
    required: true,
    type: String,
  },
});

const instanceV1Store = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const subscriptionStore = useSubscriptionV1Store();
const { t } = useI18n();

const currentUserV1 = useCurrentUserV1();

const state = reactive<LocalState>({
  showCreateDatabaseModal: false,
  syncingSchema: false,
  showFeatureModal: false,
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
  databaseStore.searchDatabaseList({
    parent: instance.value.name,
  });
});

const databaseV1List = computed(() => {
  const list = databaseStore.databaseListByInstance(instance.value.name);

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-instance",
      currentUserV1.value.userRole
    )
  ) {
    return list;
  }

  // In edge case when the user is no longer an Owner or DBA, we only want to display the database
  // belonging to the project which the user is a member of. The returned list above may contain
  // databases not meeting this criteria and we need to filter out them.
  return list.filter((db) => {
    return isMemberOfProjectV1(db.projectEntity.iamPolicy, currentUserV1.value);
  });
});

const instanceRoleList = computed(() => {
  return instanceV1Store.getInstanceRoleListByName(instance.value.name);
});

const allowEdit = computed(() => {
  return (
    instance.value.state === State.ACTIVE &&
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-instance",
      currentUserV1.value.userRole
    )
  );
});

const allowArchiveOrRestore = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-instance",
    currentUserV1.value.userRole
  );
});

const doArchive = async () => {
  await useGracefulRequest(async () => {
    await instanceV1Store.archiveInstance(instance.value);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-archived-instance-updatedinstance-name", [
        instance.value.title,
      ]),
    });
  });
};

const doRestore = async () => {
  const instanceList = instanceV1Store.activeInstanceList;
  if (subscriptionStore.instanceCount <= instanceList.length) {
    state.showFeatureModal = true;
    return;
  }
  await useGracefulRequest(async () => {
    await instanceV1Store.restoreInstance(instance.value);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-archived-instance-updatedinstance-name", [
        instance.value.title,
      ]),
    });
  });
};

const syncSchema = async () => {
  state.syncingSchema = true;
  try {
    await instanceV1Store.syncInstance(instance.value).then(() => {
      return databaseStore.searchDatabaseList({
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
