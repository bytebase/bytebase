<template>
  <div class="py-4 space-y-4">
    <ArchiveBanner v-if="instance.rowStatus == 'ARCHIVED'" />
    <BBAttention
      v-else-if="state.migrationSetupStatus != 'OK'"
      :title="attentionTitle"
      :description="attentionText"
      :actionText="attentionActionText"
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
          <div class="inline-flex space-x-2">
            <div class="text-lg leading-6 font-medium text-main">Databases</div>
          </div>
          <button
            v-if="allowEdit"
            type="button"
            class="btn-normal"
            @click.prevent="syncSchema"
          >
            Sync Now
          </button>
        </div>
        <DatabaseTable :mode="'INSTANCE'" :databaseList="databaseList" />
      </div>
      <template v-if="allowEdit">
        <template v-if="instance.rowStatus == 'NORMAL'">
          <BBButtonConfirm
            :style="'ARCHIVE'"
            :buttonText="'Archive this instance'"
            :okText="'Archive'"
            :requireConfirm="true"
            :confirmTitle="`Archive instance '${instance.name}'?`"
            :confirmDescription="'Archived instsance will not be shown on the normal interface. You can still restore later from the Archive page.'"
            @confirm="doArchive"
          />
        </template>
        <template v-else-if="instance.rowStatus == 'ARCHIVED'">
          <BBButtonConfirm
            :style="'RESTORE'"
            :buttonText="'Restore this instance'"
            :okText="'Restore'"
            :requireConfirm="true"
            :confirmTitle="`Restore instance '${instance.name}' to normal state?`"
            :confirmDescription="''"
            @confirm="doRestore"
          />
        </template>
      </template>
    </div>
  </div>

  <BBAlert
    v-if="state.showCreateMigrationSchemaModal"
    :style="'INFO'"
    :okText="'Create'"
    :title="'Create migration schema?'"
    :description="'Bytebase relies on migration schema to manage version control based schema migration for databases belonged to this instance.'"
    @ok="
      () => {
        state.showCreateMigrationSchemaModal = false;
        doCreateMigrationSchema();
      }
    "
    @cancel="state.showCreateMigrationSchemaModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { computed, reactive, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { useStore } from "vuex";
import { idFromSlug, isDBAOrOwner } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import DataSourceTable from "../components/DataSourceTable.vue";
import InstanceForm from "../components/InstanceForm.vue";
import {
  Database,
  Instance,
  InstanceMigration,
  MigrationSchemaStatus,
  SqlResultSet,
} from "../types";

interface LocalState {
  migrationSetupStatus: MigrationSchemaStatus;
  showCreateMigrationSchemaModal: boolean;
}

export default {
  name: "InstanceDetail",
  components: {
    ArchiveBanner,
    DatabaseTable,
    DataSourceTable,
    InstanceForm,
  },
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const router = useRouter();
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const state = reactive<LocalState>({
      migrationSetupStatus: "OK",
      showCreateMigrationSchemaModal: false,
    });

    const instance = computed((): Instance => {
      return store.getters["instance/instanceById"](
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
        return "Bytebase relies on migration schema to manage version control based schema migration for databases belonged to this instance.";
      } else if (state.migrationSetupStatus == "UNKNOWN") {
        return "Bytebase relies on migration schema to manage version control based schema migration for databases belonged to this instance. Please check the instance connection info is correct.";
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
      const list = store.getters["database/databaseListByInstanceId"](
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

    const allowEdit = computed(() => {
      return (
        instance.value.rowStatus == "NORMAL" &&
        isDBAOrOwner(currentUser.value.role)
      );
    });

    const doArchive = () => {
      store
        .dispatch("instance/patchInstance", {
          instanceId: instance.value.id,
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
          instanceId: instance.value.id,
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
      store
        .dispatch("instance/createMigrationSetup", instance.value.id)
        .then((resultSet: SqlResultSet) => {
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
        });
    };

    const syncSchema = () => {
      store
        .dispatch("sql/syncSchema", instance.value.id)
        .then((resultSet: SqlResultSet) => {
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
        });
    };

    return {
      state,
      attentionTitle,
      attentionText,
      attentionActionText,
      hasDataSourceFeature,
      instance,
      databaseList,
      allowEdit,
      doArchive,
      doRestore,
      doCreateMigrationSchema,
      syncSchema,
    };
  },
};
</script>
