<template>
  <div class="py-4 space-y-4">
    <ArchiveBanner v-if="instance.rowStatus == 'ARCHIVED'" />
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
            <BBButtonAdd
              v-if="instance.rowStatus == 'NORMAL'"
              @add="tryAddDatabase"
            />
          </div>
          <button type="button" class="btn-normal" @click.prevent="syncSchema">
            Sync Now
          </button>
        </div>
        <DatabaseTable :mode="'INSTANCE'" :databaseList="databaseList" />
      </div>
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
    </div>
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useStore } from "vuex";
import { idFromSlug } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import DataSourceTable from "../components/DataSourceTable.vue";
import InstanceForm from "../components/InstanceForm.vue";
import { Instance, SqlResultSet } from "../types";

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

    const hasDataSourceFeature = computed(() =>
      store.getters["plan/feature"]("bb.data-source")
    );

    const instance = computed((): Instance => {
      return store.getters["instance/instanceById"](
        idFromSlug(props.instanceSlug)
      );
    });

    const databaseList = computed(() => {
      return store.getters["database/databaseListByInstanceId"](
        instance.value.id
      );
    });

    const tryAddDatabase = () => {
      router.push({
        name: "workspace.database.create",
        query: {
          environment: instance.value.environment.id,
          instance: instance.value.id,
        },
      });
    };

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
      hasDataSourceFeature,
      instance,
      databaseList,
      tryAddDatabase,
      doArchive,
      doRestore,
      syncSchema,
    };
  },
};
</script>
