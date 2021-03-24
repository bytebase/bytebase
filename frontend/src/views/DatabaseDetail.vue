<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
    <main class="flex-1 relative pb-8 overflow-y-auto">
      <!-- Highlight Panel -->
      <div
        class="px-4 pb-4 border-b border-block-border md:flex md:items-center md:justify-between"
      >
        <div class="flex-1 min-w-0">
          <!-- Summary -->
          <div class="flex items-center">
            <div>
              <div class="flex items-center">
                <h1
                  class="pt-2 pb-2.5 text-xl font-bold leading-6 text-main truncate"
                >
                  {{ database.name }}
                </h1>
              </div>
            </div>
          </div>
          <dl
            class="flex flex-col space-y-1 md:space-y-0 md:flex-row md:flex-wrap"
          >
            <dt class="sr-only">Environment</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Environment&nbsp;-&nbsp;</span>
              <router-link to="/environment" class="normal-link">
                {{ database.instance.environment.name }}
              </router-link>
            </dd>
            <dt class="sr-only">Instance</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Instance&nbsp;-&nbsp;</span>
              <router-link
                :to="`/instance/${instanceSlug}`"
                class="normal-link"
              >
                {{ database.instance.name }}
              </router-link>
            </dd>
            <dt class="sr-only">Sync Status</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Sync status&nbsp;-&nbsp;</span>
              {{ database.syncStatus }}
            </dd>
            <dt class="sr-only">Last successful sync</dt>
            <dd class="flex items-center text-sm">
              <span class="textlabel">Last successful sync&nbsp;-&nbsp;</span>
              {{ humanizeTs(database.lastSuccessfulSyncTs) }}
            </dd>
          </dl>
        </div>
      </div>

      <div class="mt-6">
        <div
          class="max-w-6xl mx-auto px-6 space-y-6 divide-y divide-block-border"
        >
          <!-- Description list -->
          <dl class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2">
            <div class="col-span-1">
              <label for="user" class="textlabel">
                Owner <span class="text-red-600">*</span>
              </label>
              <PrincipalSelect
                class="mt-1 w-64"
                id="owner"
                name="owner"
                :disabled="!allowChangeOwner"
                :selectedId="database.ownerId"
                @select-principal-id="
                  (principalId) => {
                    updateDatabaseOwner(principalId);
                  }
                "
              />
            </div>

            <div class="col-span-1 col-start-1">
              <dt class="text-sm font-medium text-control-light">Updated</dt>
              <dd class="mt-1 text-sm text-main">
                {{ humanizeTs(database.lastUpdatedTs) }}
              </dd>
            </div>

            <div class="col-span-1">
              <dt class="text-sm font-medium text-control-light">Created</dt>
              <dd class="mt-1 text-sm text-main">
                {{ humanizeTs(database.createdTs) }}
              </dd>
            </div>
          </dl>

          <div v-if="isDBAorAbove" class="pt-6">
            <DataSourceTable :instance="instance" :database="database" />
          </div>

          <template
            v-for="(item, index) of [
              { type: 'RW', list: readWriteDataSourceList },
              { type: 'RO', list: readOnlyDataSourceList },
            ]"
            :key="index"
          >
            <div v-if="item.list.length" class="pt-6">
              <div
                v-data-source-type
                class="text-lg leading-6 font-medium text-main mb-4"
              >
                {{ item.type }}
              </div>
              <div class="space-y-4">
                <div v-for="(ds, index) of item.list" :key="index">
                  <div class="relative mb-2">
                    <div
                      class="absolute inset-0 flex items-center"
                      aria-hidden="true"
                    >
                      <div class="w-full border-t border-gray-300"></div>
                    </div>
                    <div class="relative flex justify-start">
                      <!-- Only displays the data source link for DBA and above. Since for now
                           we don't need to expose the data source concept to the end user -->
                      <router-link
                        v-if="isDBAorAbove"
                        :to="`/instance/${instanceSlug}/ds/${dataSourceSlug(
                          ds
                        )}`"
                        class="pr-3 bg-white font-medium normal-link"
                      >
                        {{ ds.name }}
                      </router-link>
                      <div v-else class="pr-3 bg-white font-medium text-main">
                        {{ "Connection " + (index + 1) }}
                      </div>
                    </div>
                  </div>
                  <DataSourceConnectionPanel :dataSource="ds" />
                </div>
              </div>
            </div>
          </template>
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import DataSourceTable from "../components/DataSourceTable.vue";
import DataSourceConnectionPanel from "../components/DataSourceConnectionPanel.vue";
import PrincipalSelect from "../components/PrincipalSelect.vue";
import { idFromSlug } from "../utils";
import { PrincipalId, DataSource } from "../types";

interface LocalState {
  editing: boolean;
  showPassword: boolean;
  editingDataSource?: DataSource;
}

export default {
  name: "DatabaseDetail",
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
    databaseSlug: {
      required: true,
      type: String,
    },
  },
  components: { DataSourceConnectionPanel, DataSourceTable, PrincipalSelect },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      editing: false,
      showPassword: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const database = computed(() => {
      return store.getters["database/databaseById"](
        idFromSlug(props.databaseSlug),
        idFromSlug(props.instanceSlug)
      );
    });

    const instance = computed(() => {
      return store.getters["instance/instanceById"](
        idFromSlug(props.instanceSlug)
      );
    });

    const isDBAorAbove = computed(() => {
      return (
        currentUser.value.role == "DBA" || currentUser.value.role == "OWNER"
      );
    });

    const allowChangeOwner = computed(() => {
      return (
        currentUser.value.id == database.value.ownerId || isDBAorAbove.value
      );
    });

    const dataSourceList = computed(() => {
      return store.getters["dataSource/dataSourceListByInstanceId"](
        instance.value.id
      ).filter((dataSource: DataSource) => {
        return (
          dataSource.databaseId == database.value.id &&
          (isDBAorAbove.value ||
            // If the current user is not DBAorAbove, we will only show the granted data source.
            dataSource.memberList.find((item) => {
              return item.principal.id == currentUser.value.id;
            }))
        );
      });
    });

    const readWriteDataSourceList = computed(() => {
      return dataSourceList.value.filter((dataSource: DataSource) => {
        return dataSource.type == "RW";
      });
    });

    const readOnlyDataSourceList = computed(() => {
      return dataSourceList.value.filter((dataSource: DataSource) => {
        return dataSource.type == "RO";
      });
    });

    const updateDatabaseOwner = (newOwnerId: PrincipalId) => {
      store
        .dispatch("database/updateOwner", {
          instanceId: instance.value.id,
          databaseId: database.value.id,
          ownerId: newOwnerId,
        })
        .then(() => {})
        .catch((error) => {
          console.error(error);
        });
    };

    return {
      state,
      database,
      instance,
      isDBAorAbove,
      allowChangeOwner,
      readWriteDataSourceList,
      readOnlyDataSourceList,
      updateDatabaseOwner,
    };
  },
};
</script>
