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
            <template v-if="isCurrentUserDBAOrOwner">
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
              <span v-database-sync-status>{{ database.syncStatus }}</span>
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
            <div class="col-span-1 w-64">
              <label for="user" class="textlabel"> Project </label>
              <ProjectSelect
                class="mt-1"
                id="project"
                name="project"
                :disabled="!allowChangeProject"
                :selectedId="database.project.id"
                @select-project-id="
                  (projectId) => {
                    updateProject(projectId);
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
                  <!-- Hide the data source detail link for now -->
                  <div v-if="false" class="relative mb-2">
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
                  <div
                    v-if="allowChangeDataSource"
                    class="flex justify-end space-x-3"
                  >
                    <template v-if="isEditingDataSource(ds)">
                      <button
                        type="button"
                        class="btn-normal"
                        @click.prevent="cancelEditDataSource"
                      >
                        Cancel
                      </button>
                      <button
                        type="button"
                        class="btn-normal"
                        :disabled="!allowSaveDataSource"
                        @click.prevent="saveEditDataSource"
                      >
                        <!-- Heroicon name: solid/save -->
                        <svg
                          class="-ml-1 mr-2 h-5 w-5 text-control-light"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                          xmlns="http://www.w3.org/2000/svg"
                        >
                          <path
                            d="M7.707 10.293a1 1 0 10-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L11 11.586V6h5a2 2 0 012 2v7a2 2 0 01-2 2H4a2 2 0 01-2-2V8a2 2 0 012-2h5v5.586l-1.293-1.293zM9 4a1 1 0 012 0v2H9V4z"
                          ></path>
                        </svg>
                        <span>Save</span>
                      </button>
                    </template>
                    <template v-else>
                      <button
                        type="button"
                        class="btn-normal"
                        @click.prevent="editDataSource(ds)"
                      >
                        <!-- Heroicon name: solid/pencil -->
                        <svg
                          class="-ml-1 mr-2 h-5 w-5 text-control-light"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                          xmlns="http://www.w3.org/2000/svg"
                        >
                          <path
                            d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
                          ></path>
                        </svg>
                        <span>Edit</span>
                      </button>
                    </template>
                  </div>
                  <DataSourceConnectionPanel
                    :editing="isEditingDataSource(ds)"
                    :dataSource="
                      isEditingDataSource(ds) ? state.editingDataSource : ds
                    "
                  />
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
import ProjectSelect from "../components/ProjectSelect.vue";
import { idFromSlug, isDBAOrOwner } from "../utils";
import { PrincipalId, DataSource, ProjectId, DataSourcePatch } from "../types";
import { cloneDeep, isEqual } from "lodash";

interface LocalState {
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
  components: {
    DataSourceConnectionPanel,
    DataSourceTable,
    PrincipalSelect,
    ProjectSelect,
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({});

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const database = computed(() => {
      return store.getters["database/databaseById"](
        idFromSlug(props.databaseSlug)
      );
    });

    const isCurrentUserDBAOrOwner = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowChangeProject = computed(() => {
      return isCurrentUserDBAOrOwner.value;
    });

    const allowChangeDataSource = computed(() => {
      return isCurrentUserDBAOrOwner.value;
    });

    const dataSourceList = computed(() => {
      return database.value.dataSourceList.filter((dataSource: DataSource) => {
        return (
          isCurrentUserDBAOrOwner.value ||
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

    const updateProject = (newProjectId: ProjectId) => {
      store
        .dispatch("database/transferProject", {
          instanceId: database.value.instance.id,
          databaseId: database.value.id,
          projectId: newProjectId,
        })
        .then((updatedDatabase) => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully transferred '${updatedDatabase.name}' to project '${updatedDatabase.project.name}'.`,
          });
        })
        .catch((error) => {
          console.error(error);
        });
    };

    const isEditingDataSource = (dataSource: DataSource) => {
      return (
        state.editingDataSource && state.editingDataSource.id == dataSource.id
      );
    };

    const allowSaveDataSource = computed(() => {
      for (const dataSource of dataSourceList.value) {
        if (dataSource.id == state.editingDataSource!.id) {
          return !isEqual(dataSource, state.editingDataSource);
        }
      }
      return false;
    });

    const editDataSource = (dataSource: DataSource) => {
      state.editingDataSource = cloneDeep(dataSource);
    };

    const cancelEditDataSource = () => {
      state.editingDataSource = undefined;
    };

    const saveEditDataSource = () => {
      const dataSourcePatch: DataSourcePatch = {
        username: state.editingDataSource?.username,
        password: state.editingDataSource?.password,
      };
      store
        .dispatch("dataSource/patchDataSource", {
          databaseId: state.editingDataSource!.database.id,
          dataSourceId: state.editingDataSource!.id,
          dataSource: dataSourcePatch,
        })
        .then(() => {
          state.editingDataSource = undefined;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    return {
      state,
      database,
      isCurrentUserDBAOrOwner,
      allowChangeProject,
      allowChangeDataSource,
      readWriteDataSourceList,
      readOnlyDataSourceList,
      updateProject,
      isEditingDataSource,
      allowSaveDataSource,
      editDataSource,
      cancelEditDataSource,
      saveEditDataSource,
    };
  },
};
</script>
