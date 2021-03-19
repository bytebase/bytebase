<template>
  <!--
  This example requires Tailwind CSS v2.0+ 
  
  This example requires some changes to your config:
  
  ```
  // tailwind.config.js
  const colors = require('tailwindcss/colors')
  
  module.exports = {
    // ...
    theme: {
      extend: {
        colors: {
          cyan: colors.cyan,
        }
      }
    },
    plugins: [
      // ...
      require('@tailwindcss/forms'),
    ]
  }
  ```
-->
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
                <input
                  v-if="state.editing"
                  required
                  ref="editNameTextField"
                  id="name"
                  name="name"
                  type="text"
                  class="textfield my-0.5 w-full"
                  v-model="state.editingDataSource.name"
                />
                <!-- Padding value is to prevent flickering when switching between edit/non-edit mode -->
                <h1
                  v-else
                  class="pt-2 pb-2.5 text-xl font-bold leading-6 text-main truncate"
                >
                  {{ dataSource.name }}
                </h1>
              </div>
              <dl
                class="flex flex-col space-y-1 sm:space-y-0 sm:flex-row sm:flex-wrap"
              >
                <dt class="sr-only">RoleType</dt>
                <dd
                  v-data-source-type
                  class="flex items-center text-sm text-control font-medium sm:mr-4"
                >
                  {{ dataSource.type }}
                </dd>
                <dt class="sr-only">Database</dt>
                <dd class="flex items-center text-sm sm:mr-4">
                  <span class="textlabel">Database&nbsp;-&nbsp;</span>
                  <router-link
                    :to="`/instance/${instanceSlug}/db/${databaseSlug(
                      database
                    )}`"
                    class="normal-link"
                  >
                    {{ database.name }}
                  </router-link>
                </dd>
                <dt class="sr-only">Instance</dt>
                <dd class="flex items-center text-sm sm:mr-4">
                  <span class="textlabel">Instance&nbsp;-&nbsp;</span>
                  <router-link
                    :to="`/instance/${instanceSlug}`"
                    class="normal-link"
                  >
                    {{ instance.name }}
                  </router-link>
                </dd>
                <dt class="sr-only">Environment</dt>
                <dd class="flex items-center text-sm">
                  <span class="textlabel">Environment&nbsp;-&nbsp;</span>
                  <router-link to="/environment" class="normal-link">
                    {{ instance.environment.name }}
                  </router-link>
                </dd>
              </dl>
            </div>
          </div>
        </div>
        <div class="mt-6 flex space-x-3 md:mt-0 md:ml-4">
          <template v-if="state.editing">
            <button
              type="button"
              class="btn-normal"
              @click.prevent="cancelEdit"
            >
              Cancel
            </button>
            <button
              type="button"
              class="btn-normal"
              :disabled="!allowSave"
              @click.prevent="saveEdit"
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
              @click.prevent="editDataSource"
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
      </div>

      <div class="mt-6">
        <div
          class="max-w-6xl mx-auto px-6 space-y-6 divide-y divide-block-border"
        >
          <!-- Description list -->
          <dl class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2">
            <div class="sm:col-span-2">
              <dt class="text-sm font-medium text-control-light">
                Connection string
              </dt>
              <dd class="mt-2 text-sm text-main">
                <div class="space-y-4">
                  <div
                    class="flex"
                    v-for="(connection, index) in connectionStringList"
                    :key="index"
                  >
                    <span
                      class="whitespace-nowrap inline-flex items-center px-3 rounded-l-md border border-l border-r-0 border-control-border bg-gray-50 text-control-light sm:text-sm"
                    >
                      {{ connection.name }}
                    </span>
                    <span
                      class="flex-1 min-w-0 block w-full inline-flex items-center px-3 py-2 border border-r border-control-border bg-gray-50 sm:text-sm"
                    >
                      {{ connection.value }}
                    </span>
                    <button
                      class="-ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light bg-gray-50 hover:bg-gray-100 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
                      @click.prevent="copyText(connection)"
                    >
                      <svg
                        class="w-6 h-6"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
                        ></path>
                      </svg>
                    </button>
                  </div>
                </div>
              </dd>
            </div>

            <div class="sm:col-span-1">
              <dt class="text-sm font-medium text-control-light">
                Username<span v-if="state.editing" class="text-red-600">*</span>
              </dt>
              <dd class="mt-1 text-sm text-main">
                <input
                  v-if="state.editing"
                  required
                  id="username"
                  type="text"
                  class="textfield"
                  v-model="state.editingDataSource.username"
                />
                <div v-else class="mt-2.5 mb-3">
                  {{ dataSource.username }}
                </div>
              </dd>
            </div>

            <div class="sm:col-span-1">
              <div class="flex items-center space-x-1">
                <dt class="text-sm font-medium text-control-light">Password</dt>
                <button
                  class="btn-icon"
                  @click.prevent="state.showPassword = !state.showPassword"
                >
                  <svg
                    v-if="state.showPassword"
                    class="w-5 h-5"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
                    ></path>
                  </svg>
                  <svg
                    v-else
                    class="w-5 h-5"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                    ></path>
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                    ></path>
                  </svg>
                </button>
              </div>
              <dd class="mt-1 text-sm text-main">
                <input
                  v-if="state.editing"
                  required
                  autocomplete="off"
                  id="password"
                  :type="state.showPassword ? 'text' : 'password'"
                  class="textfield"
                  v-model="state.editingDataSource.password"
                />
                <div v-else class="mt-2.5 mb-3">
                  <template v-if="state.showPassword">
                    {{ dataSource.password }}
                  </template>
                  <template v-else> ****** </template>
                </div>
              </dd>
            </div>

            <div class="sm:col-span-1">
              <dt class="text-sm font-medium text-control-light">Updated</dt>
              <dd class="mt-1 text-sm text-main">
                {{ humanizeTs(dataSource.lastUpdatedTs) }}
              </dd>
            </div>

            <div class="sm:col-span-1">
              <dt class="text-sm font-medium text-control-light">Created</dt>
              <dd class="mt-1 text-sm text-main">
                {{ humanizeTs(dataSource.createdTs) }}
              </dd>
            </div>
          </dl>

          <!-- Guard against dataSource.id != '-1', this could happen when we delete the data source -->
          <DataSourceMemberTable
            class="pt-6"
            v-if="dataSource.id != '-1'"
            :allowEdit="allowEdit"
            :dataSource="dataSource"
          />

          <div class="pt-4 flex justify-start">
            <BBButtonTrash
              v-if="allowEdit"
              :buttonText="'Delete this entire data source'"
              :requireConfirm="true"
              :confirmTitle="`Are you sure to delete '${dataSource.name}'?`"
              :confirmDescription="'All existing users using this data source to connect the database will fail. You cannot undo this action.'"
              @confirm="doDelete"
            />
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, nextTick, reactive, ref } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import DataSourceMemberTable from "../components/DataSourceMemberTable.vue";
import { idFromSlug } from "../utils";
import { ALL_DATABASE_NAME, DataSource } from "../types";

type Connection = {
  name: string;
  value: string;
};

interface LocalState {
  editing: boolean;
  showPassword: boolean;
  editingDataSource?: DataSource;
}

export default {
  name: "DataSourceDetail",
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
    dataSourceSlug: {
      required: true,
      type: String,
    },
  },
  components: { DataSourceMemberTable },
  setup(props, ctx) {
    const editNameTextField = ref();

    const store = useStore();
    const router = useRouter();

    const instanceId = idFromSlug(props.instanceSlug);
    const dataSourceId = idFromSlug(props.dataSourceSlug);

    const state = reactive<LocalState>({
      editing: false,
      showPassword: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const dataSource = computed(() => {
      return store.getters["dataSource/dataSourceById"](
        dataSourceId,
        instanceId
      );
    });

    const instance = computed(() => {
      return store.getters["instance/instanceById"](instanceId);
    });

    const database = computed(() => {
      return store.getters["database/databaseById"](
        dataSource.value.databaseId,
        instanceId
      );
    });

    const connectionStringList = computed<Connection[]>(() => {
      // If host starts with "/", we assume it's a local socket.
      const isSocket = instance.value.host.startsWith("/");
      const cliOptionList = isSocket
        ? [`mysql -S ${instance.value.host}`]
        : [`mysql -h ${instance.value.host}`];
      if (instance.value.port) {
        cliOptionList.push(`-P ${instance.value.port}`);
      }
      if (database.value.name != ALL_DATABASE_NAME) {
        cliOptionList.push(`-D ${database.value.name}`);
      }
      if (dataSource.value.username) {
        cliOptionList.push(`-u ${dataSource.value.username}`);
      }
      if (dataSource.value.password) {
        if (state.showPassword) {
          cliOptionList.push(`-p${dataSource.value.password}`);
        } else {
          cliOptionList.push(`-p`);
        }
      }

      let jdbcString = `JDBC can't connect to socket ${instance.value.host} `;
      if (!isSocket) {
        jdbcString = `jdbc:mysql://${instance.value.host}`;
        if (instance.value.port) {
          jdbcString += `:${instance.value.port}`;
        }
        if (database.value.name != ALL_DATABASE_NAME) {
          jdbcString += `/${database.value.name}`;
        }
        const optionList = [];
        if (dataSource.value.username) {
          optionList.push(`user=${dataSource.value.username}`);
        }
        if (dataSource.value.password) {
          if (state.showPassword) {
            optionList.push(`password=${dataSource.value.password}`);
          } else {
            optionList.push(`password=******`);
          }
        }
        if (optionList.length > 0) {
          jdbcString += `&${optionList.join("&")}`;
        }
      }

      return [
        { name: "CLI", value: cliOptionList.join(" ") },
        { name: "JDBC", value: jdbcString },
      ];
    });

    const allowEdit = computed(() => {
      return (
        currentUser.value.role == "OWNER" || currentUser.value.role == "DBA"
      );
    });

    const allowSave = computed(() => {
      return (
        state.editingDataSource!.name &&
        !isEqual(dataSource.value, state.editingDataSource)
      );
    });

    const editDataSource = () => {
      state.editingDataSource = cloneDeep(dataSource.value);
      state.editing = true;

      nextTick(() => editNameTextField.value.focus());
    };

    const cancelEdit = () => {
      state.editingDataSource = undefined;
      state.editing = false;
    };

    const saveEdit = () => {
      store
        .dispatch("dataSource/patchDataSource", {
          instanceId,
          dataSource: state.editingDataSource,
        })
        .then(() => {
          state.editingDataSource = undefined;
          state.editing = false;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doDelete = () => {
      const name = dataSource.value.name;
      store
        .dispatch("dataSource/deleteDataSourceById", {
          instanceId,
          dataSourceId,
        })
        .then(() => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully deleted data source '${name}'.`,
          });
          router.push(`/instance/${props.instanceSlug}`);
        })
        .catch((error) => {
          console.error(error);
        });
    };

    const copyText = (connection: Connection) => {
      toClipboard(connection.value).then(() => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "INFO",
          title: `${connection.name} string copied to clipboard.`,
        });
      });
    };

    return {
      editNameTextField,
      state,
      dataSource,
      instance,
      database,
      connectionStringList,
      allowEdit,
      allowSave,
      editDataSource,
      cancelEdit,
      saveEdit,
      doDelete,
      copyText,
    };
  },
};
</script>
