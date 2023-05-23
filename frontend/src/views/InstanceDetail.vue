<template>
  <div class="py-4 space-y-4">
    <ArchiveBanner v-if="instanceV1.state === State.DELETED" />
    <BBAttention
      v-else-if="state.migrationSetupStatus != 'OK'"
      :style="'WARN'"
      :title="attentionTitle"
      :description="attentionText"
      :action-text="attentionActionText"
      @click-action="state.showCreateMigrationSchemaModal = true"
    />
    <div class="px-6 space-y-6">
      <!-- <InstanceForm :instance="instance" /> -->
      <div class="text-red-500">should be InstanceForm</div>
      <div class="text-read-500">{{ instanceV1 }}</div>
      <div
        v-if="hasDataSourceFeature"
        class="py-6 space-y-4 border-t divide-control-border"
      >
        <!-- <DataSourceTable :instance="instance" /> -->
        <div class="text-red-500">Should be DataSourceTable</div>
      </div>
      <div v-else>
        <div class="mb-4 flex items-center justify-between">
          <BBTabFilter
            :tab-item-list="tabItemList"
            :selected-index="state.selectedIndex"
            @select-index="
              (index: number) => {
                state.selectedIndex = index;
              }
            "
          />
          <div class="flex items-center space-x-4">
            <div>
              <BBSpin
                v-if="state.syncingSchema"
                :title="$t('instance.syncing')"
              />
            </div>
            <button
              v-if="allowEdit"
              :disabled="state.syncingSchema"
              type="button"
              class="btn-normal"
              @click.prevent="syncSchema"
            >
              {{ $t("common.sync-now") }}
            </button>
            <button
              v-if="
                instanceV1.state === State.ACTIVE &&
                instanceV1HasCreateDatabase(instanceV1)
              "
              type="button"
              class="btn-primary"
              @click.prevent="createDatabase"
            >
              {{ $t("instance.new-database") }}
            </button>
          </div>
        </div>
        <div v-if="state.selectedIndex == DATABASE_TAB">
          <DatabaseTable
            mode="INSTANCE"
            :scroll-on-page-change="false"
            :database-list="databaseList"
          />
        </div>
        <InstanceUserTable
          v-else-if="state.selectedIndex == USER_TAB"
          :instance-user-list="instanceUserList"
        />
      </div>
      <template v-if="allowArchiveOrRestore">
        <template v-if="instanceV1.state === State.ACTIVE">
          <BBButtonConfirm
            :style="'ARCHIVE'"
            :button-text="$t('instance.archive-this-instance')"
            :ok-text="$t('common.archive')"
            :require-confirm="true"
            :confirm-title="
              $t('instance.archive-instance-instance-name', [instanceV1.title])
            "
            :confirm-description="
              $t(
                'instance.archived-instances-will-not-be-shown-on-the-normal-interface-you-can-still-restore-later-from-the-archive-page'
              )
            "
            @confirm="doArchive"
          />
        </template>
        <template v-else-if="instanceV1.state === State.DELETED">
          <BBButtonConfirm
            :style="'RESTORE'"
            :button-text="$t('instance.restore-this-instance')"
            :ok-text="$t('instance.restore')"
            :require-confirm="true"
            :confirm-title="
              $t('instance.restore-instance-instance-name-to-normal-state', [
                instanceV1.title,
              ])
            "
            :confirm-description="''"
            @confirm="doRestore"
          />
        </template>
      </template>
    </div>
  </div>

  <BBAlert
    v-if="state.showCreateMigrationSchemaModal"
    :style="'INFO'"
    :ok-text="$t('common.create')"
    :title="$t('instance.create-migration-schema') + '?'"
    :description="
      $t(
        'instance.bytebase-relies-on-migration-schema-to-manage-gitops-based-schema-migration-for-databases-belonged-to-this-instance'
      )
    "
    :in-progress="state.creatingMigrationSchema"
    @ok="
      () => {
        doCreateMigrationSchema();
      }
    "
    @cancel="state.showCreateMigrationSchemaModal = false"
  ></BBAlert>

  <BBModal
    v-if="state.showCreateDatabaseModal"
    :title="$t('quick-action.create-db')"
    @close="state.showCreateDatabaseModal = false"
  >
    <CreateDatabasePrepForm
      :environment-id="environment?.uid"
      :instance-id="instanceId"
      @dismiss="state.showCreateDatabaseModal = false"
    />
  </BBModal>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.instance-count"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import {
  idFromSlug,
  hasWorkspacePermissionV1,
  instanceV1HasCreateDatabase,
  isMemberOfProjectV1,
} from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import DataSourceTable from "../components/DataSourceTable.vue";
import InstanceUserTable from "../components/InstanceUserTable.vue";
import InstanceForm from "../components/InstanceForm.vue";
import CreateDatabasePrepForm from "../components/CreateDatabasePrepForm.vue";
import {
  Database,
  Instance,
  InstanceMigration,
  MigrationSchemaStatus,
  SQLResultSet,
} from "../types";
import { BBTabFilterItem } from "../bbkit/types";
import { useI18n } from "vue-i18n";
import {
  featureToRef,
  pushNotification,
  useDatabaseStore,
  useInstanceStore,
  useSubscriptionStore,
  useSQLStore,
  useDBSchemaStore,
  useProjectV1Store,
  useCurrentUserV1,
  useInstanceV1Store,
  useEnvironmentV1Store,
} from "@/store";
import { State } from "@/types/proto/v1/common";

const DATABASE_TAB = 0;
const USER_TAB = 1;

interface LocalState {
  selectedIndex: number;
  migrationSetupStatus: MigrationSchemaStatus;
  showCreateMigrationSchemaModal: boolean;
  creatingMigrationSchema: boolean;
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

const instanceStore = useInstanceStore();
const instanceV1Store = useInstanceV1Store();
const subscriptionStore = useSubscriptionStore();
const { t } = useI18n();

const currentUserV1 = useCurrentUserV1();
const sqlStore = useSQLStore();

const state = reactive<LocalState>({
  selectedIndex: DATABASE_TAB,
  migrationSetupStatus: "OK",
  showCreateMigrationSchemaModal: false,
  creatingMigrationSchema: false,
  showCreateDatabaseModal: false,
  syncingSchema: false,
  showFeatureModal: false,
});

// const instance = computed((): Instance => {
//   return instanceStore.getInstanceById(idFromSlug(props.instanceSlug));
// });
const instanceId = computed(() => {
  return idFromSlug(props.instanceSlug);
});
const instanceV1 = computed(() => {
  return instanceV1Store.getInstanceByUID(String(instanceId.value));
});
const environment = computed(() => {
  return useEnvironmentV1Store().getEnvironmentByName(
    instanceV1.value.environment
  );
});

watchEffect(() => console.log(JSON.stringify(instanceV1.value)));

const checkMigrationSetup = () => {
  instanceStore
    .checkMigrationSetup(instanceId.value)
    .then((migration: InstanceMigration) => {
      state.migrationSetupStatus = migration.status;
    });
};

const prepareMigrationSchemaStatus = () => {
  checkMigrationSetup();
};
watchEffect(prepareMigrationSchemaStatus);

const attentionTitle = computed((): string => {
  if (state.migrationSetupStatus == "NOT_EXIST") {
    return t("instance.missing-migration-schema");
  } else if (state.migrationSetupStatus == "UNKNOWN") {
    return t("instance.unable-to-connect-instance-to-check-migration-schema");
  }
  return "";
});

const attentionText = computed((): string => {
  if (state.migrationSetupStatus == "NOT_EXIST") {
    return (
      t(
        "instance.bytebase-relies-on-migration-schema-to-manage-gitops-based-schema-migration-for-databases-belonged-to-this-instance"
      ) +
      (hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-instance",
        currentUserV1.value.userRole
      )
        ? ""
        : " " + t("instance.please-contact-your-dba-to-configure-it"))
    );
  } else if (state.migrationSetupStatus == "UNKNOWN") {
    return (
      t(
        "instance.bytebase-relies-on-migration-schema-to-manage-gitops-based-schema-migration-for-databases-belonged-to-this-instance"
      ) +
      (hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-instance",
        currentUserV1.value.userRole
      )
        ? " " +
          t("instance.please-check-the-instance-connection-info-is-correct")
        : " " + t("instance.please-contact-your-dba-to-configure-it"))
    );
  }
  return "";
});

const attentionActionText = computed((): string => {
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-instance",
      currentUserV1.value.userRole
    )
  ) {
    if (state.migrationSetupStatus == "NOT_EXIST") {
      return t("instance.create-migration-schema");
    } else if (state.migrationSetupStatus == "UNKNOWN") {
      return "";
    }
  }
  return "";
});

const hasDataSourceFeature = featureToRef("bb.feature.data-source");

const databaseList = computed(() => {
  const list = useDatabaseStore().getDatabaseListByInstanceId(instanceId.value);

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
  const filteredList: Database[] = [];
  for (const database of list) {
    const projectV1 = useProjectV1Store().getProjectByUID(
      String(database.project.id)
    );
    if (isMemberOfProjectV1(projectV1.iamPolicy, currentUserV1.value)) {
      filteredList.push(database);
    }
  }

  return filteredList;
});

const instanceUserList = computed(() => {
  return instanceStore.getInstanceUserListById(instanceId.value);
});

const allowEdit = computed(() => {
  return (
    instanceV1.value.state === State.ACTIVE &&
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

const tabItemList = computed((): BBTabFilterItem[] => {
  return [
    {
      title: t("common.databases"),
      alert: false,
    },
    {
      title: t("instance.users"),
      alert: false,
    },
  ];
});

const doArchive = () => {
  instanceStore
    .patchInstance({
      instanceId: instanceId.value,
      instancePatch: {
        rowStatus: "ARCHIVED",
      },
    })
    .then((updatedInstance) => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "instance.successfully-archived-instance-updatedinstance-name",
          [updatedInstance.name]
        ),
      });
    });
};

const doRestore = () => {
  const instanceList = instanceStore.getInstanceList(["NORMAL"]);
  if (subscriptionStore.instanceCount <= instanceList.length) {
    state.showFeatureModal = true;
    return;
  }
  instanceStore
    .patchInstance({
      instanceId: instanceId.value,
      instancePatch: {
        rowStatus: "NORMAL",
      },
    })
    .then((updatedInstance) => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "instance.successfully-restored-instance-updatedinstance-name",
          [updatedInstance.name]
        ),
      });
    });
};

const doCreateMigrationSchema = () => {
  state.creatingMigrationSchema = true;
  instanceStore
    .createMigrationSetup(instanceId.value)
    .then((resultSet: SQLResultSet) => {
      state.creatingMigrationSchema = false;
      if (resultSet.error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t(
            "instance.failed-to-create-migration-schema-for-instance-instance-value-name",
            [instanceV1.value.title]
          ),
          description: resultSet.error,
        });
      } else {
        checkMigrationSetup();
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t(
            "instance.successfully-created-migration-schema-for-instance-value-name",
            [instanceV1.value.title]
          ),
        });
      }
      state.showCreateMigrationSchemaModal = false;
    });
};

const syncSchema = () => {
  state.syncingSchema = true;
  sqlStore
    .syncSchema(instanceId.value)
    .then((resultSet: SQLResultSet) => {
      state.syncingSchema = false;
      if (resultSet.error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t(
            "instance.failed-to-sync-schema-for-instance-instance-value-name",
            [instanceV1.value.title]
          ),
          description: resultSet.error,
        });
      } else {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t(
            "instance.successfully-synced-schema-for-instance-instance-value-name",
            [instanceV1.value.title]
          ),
          description: resultSet.error,
        });
      }

      // Clear the db schema metadata cache entities.
      // So we will re-fetch new values when needed.
      const dbSchemaStore = useDBSchemaStore();
      const databaseList = useDatabaseStore().getDatabaseListByInstanceId(
        instanceId.value
      );
      databaseList.forEach((database) =>
        dbSchemaStore.removeCacheByDatabaseId(database.id)
      );
    })
    .catch(() => {
      state.syncingSchema = false;
    });
};

const createDatabase = () => {
  state.showCreateDatabaseModal = true;
};
</script>
