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
              <router-link
                :to="`/environment/${environmentSlug(
                  database.instance.environment
                )}`"
                class="normal-link"
              >
                {{ environmentName(database.instance.environment) }}
              </router-link>
            </dd>
            <template v-if="isCurrentUserDBA">
              <dt class="sr-only">Instance</dt>
              <dd class="flex items-center text-sm md:mr-4">
                <span class="textlabel">Instance&nbsp;-&nbsp;</span>
                <router-link
                  :to="`/instance/${instanceSlug(database.instance)}`"
                  class="normal-link"
                >
                  {{ instanceName(database.instance) }}
                </router-link>
              </dd>
            </template>
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

          <!-- Hide data source list for now, as we don't allow adding new data source after creating the database. -->
          <div v-if="false" class="pt-6">
            <DataSourceTable
              :instance="database.instance"
              :database="database"
            />
          </div>

          <template
            v-for="(item, index) of [
              { type: 'RW', list: readWriteDataSourceList },
              { type: 'RO', list: readOnlyDataSourceList },
            ]"
            :key="index"
          >
            <div v-if="item.list.length" class="pt-6">
              <div class="text-lg leading-6 font-medium text-main mb-4">
                <span v-data-source-type>{{ item.type }}</span>
              </div>
              <div class="space-y-4">
                <div v-for="(ds, index) of item.list" :key="index">
                  <!-- Only displays the data source link for DBA. Since for now
                  we don't need to expose the data source concept to the end user -->
                  <div v-if="isCurrentUserDBA" class="relative mb-2">
                    <div
                      class="absolute inset-0 flex items-center"
                      aria-hidden="true"
                    >
                      <div class="w-full border-t border-gray-300"></div>
                    </div>
                    <div class="relative flex justify-start">
                      <router-link
                        :to="`/db/${databaseSlug}/datasource/${dataSourceSlug(
                          ds
                        )}`"
                        class="pr-3 bg-white font-medium normal-link"
                      >
                        {{ ds.name }}
                      </router-link>
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
import { computed, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import DataSourceTable from "../components/DataSourceTable.vue";
import DataSourceConnectionPanel from "../components/DataSourceConnectionPanel.vue";
import PrincipalSelect from "../components/PrincipalSelect.vue";
import { idFromSlug, isDBA } from "../utils";
import { PrincipalId, DataSource } from "../types";

interface LocalState {
  editing: boolean;
  showPassword: boolean;
  editingDataSource?: DataSource;
}

export default {
  name: "DatabaseDetail",
  props: {
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
        idFromSlug(props.databaseSlug)
      );
    });

    const isCurrentUserDBA = computed((): boolean => {
      return isDBA(currentUser.value.role);
    });

    const allowChangeOwner = computed(() => {
      return (
        currentUser.value.id == database.value.ownerId || isCurrentUserDBA.value
      );
    });

    const dataSourceList = computed(() => {
      return database.value.dataSourceList.filter((dataSource: DataSource) => {
        return (
          isCurrentUserDBA.value ||
          // If the current user is not DBA, we will only show the granted data source.
          dataSource.memberList.find((item) => {
            return item.principal.id == currentUser.value.id;
          })
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
          instanceId: database.value.instance.id,
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
      isCurrentUserDBA,
      allowChangeOwner,
      readWriteDataSourceList,
      readOnlyDataSourceList,
      updateDatabaseOwner,
    };
  },
};
</script>
