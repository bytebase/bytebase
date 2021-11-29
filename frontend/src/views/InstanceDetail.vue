<template>
  <div class="py-4 space-y-4">
    <ArchiveBanner v-if="instance.rowStatus == 'ARCHIVED'" />
    <BBAttention
      v-else-if="state.migrationSetupStatus != 'OK'"
      :style="'WARN'"
      :title="attentionTitle"
      :description="attentionText"
      :action-text="attentionActionText"
      @click-action="state.showCreateMigrationSchemaModal = true"
    />
    <div class="px-6 space-y-6">
      <InstanceForm :create="false" :instance="instance" />
      <div
        v-if="hasDataSourceFeature"
        class="py-6 space-y-4 border-t divide-control-border"
      >
        <DataSourceTable :instance="instance" />
      </div>
      <div v-else>
        <div class="mb-4 flex items-center justify-between">
          <BBTabFilter
            :tab-item-list="tabItemList"
            :selected-index="state.selectedIndex"
            @select-index="
              (index) => {
                state.selectedIndex = index;
              }
            "
          />
          <div class="flex items-center space-x-4">
            <div>
              <BBSpin v-if="state.syncingSchema" :title="'Syncing ...'" />
            </div>
            <button
              v-if="allowEdit"
              type="button"
              class="btn-normal"
              @click.prevent="syncSchema"
            >
              Sync Now
            </button>
            <button
              v-if="instance.rowStatus == 'NORMAL'"
              type="button"
              class="btn-primary"
              @click.prevent="createDatabase"
            >
              New Database
            </button>
          </div>
        </div>
        <DatabaseTable
          v-if="state.selectedIndex == DATABASE_TAB"
          :mode="'INSTANCE'"
          :database-list="databaseList"
        />
        <InstanceUserTable
          v-else-if="state.selectedIndex == USER_TAB"
          :instance-user-list="instanceUserList"
        />
      </div>
      <template v-if="allowEdit">
        <template v-if="instance.rowStatus == 'NORMAL'">
          <BBButtonConfirm
            :style="'ARCHIVE'"
            :button-text="'Archive this instance'"
            :ok-text="'Archive'"
            :require-confirm="true"
            :confirm-title="`Archive instance '${instance.name}'?`"
            :confirm-description="'Archived instances will not be shown on the normal interface. You can still restore later from the Archive page.'"
            @confirm="doArchive"
          />
        </template>
        <template v-else-if="instance.rowStatus == 'ARCHIVED'">
          <BBButtonConfirm
            :style="'RESTORE'"
            :button-text="'Restore this instance'"
            :ok-text="'Restore'"
            :require-confirm="true"
            :confirm-title="`Restore instance '${instance.name}' to normal state?`"
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
    :ok-text="'Create'"
    :title="'Create migration schema?'"
    :description="'Bytebase relies on migration schema to manage version control based schema migration for databases belonged to this instance.'"
    :in-progress="state.creatingMigrationSchema"
    @ok="
      () => {
        doCreateMigrationSchema();
      }
    "
    @cancel="state.showCreateMigrationSchemaModal = false"
  >
  </BBAlert>

  <BBModal
    v-if="state.showCreateDatabaseModal"
    :title="'Create database'"
    @close="state.showCreateDatabaseModal = false"
  >
    <!-- eslint-disable vue/attribute-hyphenation -->
    <CreateDatabasePrepForm
      :environmentID="instance.environment.id"
      :instanceID="instance.id"
      @dismiss="state.showCreateDatabaseModal = false"
    />
  </BBModal>
</template>

<script lang="ts">
import { computed, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import { idFromSlug, isDBAOrOwner } from "../utils";
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
  SqlResultSet,
} from "../types";
import { BBTabFilterItem } from "../bbkit/types";

const DATABASE_TAB = 0;
const USER_TAB = 1;

interface LocalState {
  selectedIndex: number;
  migrationSetupStatus: MigrationSchemaStatus;
  showCreateMigrationSchemaModal: boolean;
  creatingMigrationSchema: boolean;
  showCreateDatabaseModal: boolean;
  syncingSchema: boolean;
}

export default {
  name: "InstanceDetail",
  components: {
    ArchiveBanner,
    DatabaseTable,
    DataSourceTable,
    InstanceUserTable,
    InstanceForm,
    CreateDatabasePrepForm,
  },
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
  },
  setup(props) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const state = reactive<LocalState>({
      selectedIndex: DATABASE_TAB,
      migrationSetupStatus: "OK",
      showCreateMigrationSchemaModal: false,
      creatingMigrationSchema: false,
      showCreateDatabaseModal: false,
      syncingSchema: false,
    });

    const instance = computed((): Instance => {
      return store.getters["instance/instanceByID"](
        idFromSlug(props.instanceSlug)
      );
    });

    const checkMigrationSetup = () => {
      store
        .dispatch("instance/checkMigrationSetup", instance.value.id)
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
        return "Missing migration schema";
      } else if (state.migrationSetupStatus == "UNKNOWN") {
        return "Unable to connect instance to check migration schema";
      }
      return "";
    });

    const attentionText = computed((): string => {
      if (state.migrationSetupStatus == "NOT_EXIST") {
        return (
          "Bytebase relies on migration schema to manage version control based schema migration for databases belonged to this instance." +
          (isDBAOrOwner(currentUser.value.role)
            ? ""
            : " Please contact your DBA to configure it.")
        );
      } else if (state.migrationSetupStatus == "UNKNOWN") {
        return (
          "Bytebase relies on migration schema to manage version control based schema migration for databases belonged to this instance." +
          (isDBAOrOwner(currentUser.value.role)
            ? " Please check the instance connection info is correct."
            : " Please contact your DBA to configure it.")
        );
      }
      return "";
    });

    const attentionActionText = computed((): string => {
      if (isDBAOrOwner(currentUser.value.role)) {
        if (state.migrationSetupStatus == "NOT_EXIST") {
          return "Create migration schema";
        } else if (state.migrationSetupStatus == "UNKNOWN") {
          return "";
        }
      }
      return "";
    });

    const hasDataSourceFeature = computed(() =>
      store.getters["plan/feature"]("bb.data-source")
    );

    const databaseList = computed(() => {
      const list = store.getters["database/databaseListByInstanceID"](
        instance.value.id
      );
      if (isDBAOrOwner(currentUser.value.role)) {
        return list;
      }

      // In edge case when the user is no longer an Owner or DBA, we only want to display the database
      // belonging to the project which the user is a member of. The returned list above may contain
      // databases not meeting this criteria and we need to filter out them.
      const filteredList: Database[] = [];
      for (const database of list) {
        for (const member of database.project.memberList) {
          if (member.principal.id == currentUser.value.id) {
            filteredList.push(database);
            break;
          }
        }
      }
      return filteredList;
    });

    const instanceUserList = computed(() => {
      return store.getters["instance/instanceUserListByID"](instance.value.id);
    });

    const allowEdit = computed(() => {
      return (
        instance.value.rowStatus == "NORMAL" &&
        isDBAOrOwner(currentUser.value.role)
      );
    });

    const tabItemList = computed((): BBTabFilterItem[] => {
      return [
        {
          title: "Databases",
          alert: false,
        },
        {
          title: "Users",
          alert: false,
        },
      ];
    });

    const doArchive = () => {
      store
        .dispatch("instance/patchInstance", {
          instanceID: instance.value.id,
          instancePatch: {
            rowStatus: "ARCHIVED",
          },
        })
        .then((updatedInstance) => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully archived instance '${updatedInstance.name}'.`,
          });
        });
    };

    const doRestore = () => {
      store
        .dispatch("instance/patchInstance", {
          instanceID: instance.value.id,
          instancePatch: {
            rowStatus: "NORMAL",
          },
        })
        .then((updatedInstance) => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully restored instance '${updatedInstance.name}'.`,
          });
        });
    };

    const doCreateMigrationSchema = () => {
      state.creatingMigrationSchema = true;
      store
        .dispatch("instance/createMigrationSetup", instance.value.id)
        .then((resultSet: SqlResultSet) => {
          state.creatingMigrationSchema = false;
          if (resultSet.error) {
            store.dispatch("notification/pushNotification", {
              module: "bytebase",
              style: "CRITICAL",
              title: `Failed to create migration schema for instance '${instance.value.name}'.`,
              description: resultSet.error,
            });
          } else {
            checkMigrationSetup();
            store.dispatch("notification/pushNotification", {
              module: "bytebase",
              style: "SUCCESS",
              title: `Successfully created migration schema for '${instance.value.name}'.`,
            });
          }
          state.showCreateMigrationSchemaModal = false;
        });
    };

    const syncSchema = () => {
      state.syncingSchema = true;
      store
        .dispatch("sql/syncSchema", instance.value.id)
        .then((resultSet: SqlResultSet) => {
          state.syncingSchema = false;
          if (resultSet.error) {
            store.dispatch("notification/pushNotification", {
              module: "bytebase",
              style: "CRITICAL",
              title: `Failed to sync schema for instance '${instance.value.name}'.`,
              description: resultSet.error,
            });
          } else {
            store.dispatch("notification/pushNotification", {
              module: "bytebase",
              style: "SUCCESS",
              title: `Successfully synced schema for instance '${instance.value.name}'.`,
              description: resultSet.error,
            });
          }
        })
        .catch(() => {
          state.syncingSchema = false;
        });
    };

    const createDatabase = () => {
      state.showCreateDatabaseModal = true;
    };

    return {
      DATABASE_TAB,
      USER_TAB,
      state,
      attentionTitle,
      attentionText,
      attentionActionText,
      hasDataSourceFeature,
      instance,
      databaseList,
      instanceUserList,
      allowEdit,
      tabItemList,
      doArchive,
      doRestore,
      doCreateMigrationSchema,
      syncSchema,
      createDatabase,
    };
  },
};
</script>
