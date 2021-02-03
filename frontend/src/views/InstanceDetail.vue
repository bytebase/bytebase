<style scoped>
/*  Removed the ticker in the number field  */
input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

/* Firefox */
input[type="number"] {
  -moz-appearance: textfield;
}
</style>

<template>
  <form class="px-4 space-y-6 divide-y divide-control-border">
    <!-- Instance Name -->
    <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
      <div class="sm:col-span-2">
        <label for="name" class="text-sm font-medium text-gray-700">
          Instance Name <span style="color: red">*</span>
        </label>
        <input
          required
          id="name"
          name="name"
          type="text"
          class="shadow-sm mt-1 focus:ring-accent block w-full sm:text-sm border-control-border rounded-md"
          :value="state.instance.attributes.name"
          @input="updateInstance('name', $event.target.value)"
        />
      </div>

      <div class="sm:col-span-2 sm:col-start-1">
        <label for="environment" class="text-sm font-medium text-gray-700">
          Environment <span style="color: red">*</span>
        </label>
        <EnvironmentSelect
          id="environment"
          name="environment"
          :selectedId="state.instance.attributes.environmentId"
          @select-environment-id="
            (environmentId) => {
              updateInstance('environmentId', environmentId);
            }
          "
        />
      </div>

      <div class="sm:col-span-5">
        <label for="host" class="block text-sm font-medium text-gray-700">
          Host or Socket <span style="color: red">*</span>
        </label>
        <div class="mt-1">
          <input
            required
            type="text"
            id="host"
            name="host"
            placeholder="e.g. 127.0.0.1 | localhost | /tmp/mysql.sock"
            class="shadow-sm focus:ring-accent block w-full sm:text-sm border-control-border rounded-md"
            :value="state.instance.attributes.host"
            @input="updateInstance('host', $event.target.value)"
          />
        </div>
      </div>

      <div class="sm:col-span-1">
        <label for="port" class="block text-sm font-medium text-gray-700">
          Port
        </label>
        <div class="mt-1">
          <input
            type="number"
            id="port"
            name="port"
            placeholder="3306"
            class="shadow-sm focus:ring-accent block w-full sm:text-sm border-control-border rounded-md"
            :value="state.instance.attributes.port"
            @input="updateInstance('port', $event.target.value)"
          />
        </div>
      </div>

      <div class="sm:col-span-6 sm:col-start-1">
        <label
          for="externallink"
          class="inline-flex text-sm font-medium text-gray-700"
        >
          <span class="">External Link</span>
          <button
            class="btn-icon"
            :disabled="
              state.instance.attributes.externalLink?.trim().length == 0
            "
            @click.prevent="
              window.open(
                urlfy(state.instance.attributes.externalLink),
                '_blank'
              )
            "
          >
            <svg
              class="ml-1 w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
              ></path>
            </svg>
          </button>
        </label>
        <input
          required
          id="externallink"
          name="externallink"
          type="text"
          class="shadow-sm mt-1 focus:ring-accent block w-full sm:text-sm border-control-border rounded-md"
          :value="state.instance.attributes.externalLink"
          @input="updateInstance('externalLink', $event.target.value)"
        />
      </div>
    </div>
    <!-- Datasource Info -->
    <div class="pt-6">
      <div class="flex justify-between">
        <div>
          <h3 class="text-lg leading-6 font-medium text-gray-900">
            Admin Data Source Info
          </h3>
          <p class="mt-1 text-sm text-gray-500">
            This data source usually has admin privilege and is the default data
            source used by the instance owner (e.g. DBA) connecting to the
            instance to perform administrative operations.
          </p>
        </div>
      </div>
      <div class="mt-4">
        <button type="button" class="btn-normal">Test Connection</button>
      </div>
      <div class="pt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="username" class="block text-sm font-medium text-gray-700">
            Username <span style="color: red">*</span>
          </label>
          <div class="mt-1">
            <!-- For mysql, username can be empty indicating anonymous user. 
            But it's a very bad practice to use anonymous user for admin operation,
            thus we make it REQUIRED here. -->
            <input
              required
              id="username"
              name="username"
              type="text"
              class="shadow-sm focus:ring-accent block w-full sm:text-sm border-control-border rounded-md"
              :value="state.adminDataSource.attributes.username"
              @input="updateDataSource('username', $event.target.value)"
            />
          </div>
        </div>

        <div class="sm:col-span-2 sm:col-start-1">
          <label for="password" class="block text-sm font-medium text-gray-700">
            Password
          </label>
          <div class="mt-1 inline-flex">
            <input
              id="password"
              name="password"
              :type="state.showPassword ? 'text' : 'password'"
              class="shadow-sm focus:ring-accent block w-full sm:text-sm border-control-border rounded-md"
              :value="state.adminDataSource.attributes.password"
              @input="updateDataSource('password', $event.target.value)"
            />
            <div class="ml-2 flex items-center">
              <button
                class="btn-icon"
                @click.prevent="state.showPassword = !state.showPassword"
              >
                <svg
                  v-if="state.showPassword"
                  class="w-7 h-7"
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
                  class="w-7 h-7"
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
          </div>
        </div>
      </div>
    </div>
    <!-- Action Button Group -->
    <div>
      <!-- Create button group -->
      <div v-if="state.new" class="flex justify-end pt-5">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="goBack"
        >
          Cancel
        </button>
        <button
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          @click.prevent="doCreate(state.instance, state.adminDataSource)"
        >
          Create
        </button>
      </div>
      <!-- Update button group -->
      <div v-else class="flex justify-between pt-5">
        <button
          type="button"
          class="btn-danger py-2 px-4"
          @click.prevent="state.showDeleteModal = true"
        >
          Delete
        </button>
        <div>
          <button
            type="button"
            class="btn-normal py-2 px-4"
            :disabled="!valueChanged"
            @click.prevent="revertInstanceAndDataSource"
          >
            Revert
          </button>
          <button
            type="button"
            class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
            :disabled="!valueChanged"
            @click.prevent="doUpdate(state.instance, state.adminDataSource)"
          >
            Update
          </button>
        </div>
      </div>
    </div>
  </form>
  <BBAlert
    :showing="state.showDeleteModal"
    :style="'critical'"
    :okText="'Delete'"
    :title="'Delete instance \'' + state.instance.attributes.name + '\' ?'"
    @ok="
      () => {
        state.showDeleteModal = false;
        doDelete();
      }
    "
    @cancel="state.showDeleteModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { computed, onMounted, reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import isEqual from "lodash-es/isEqual";
import { urlfy } from "../utils";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import { Instance, NewInstance, DataSource, NewDataSource } from "../types";

interface LocalState {
  new: boolean;
  originalInstance?: Instance;
  instance?: Instance | NewInstance;
  originalAdminDataSource?: DataSource;
  adminDataSource?: DataSource | NewDataSource;
  showDeleteModal: boolean;
  showPassword: boolean;
}

export default {
  name: "InstanceDetail",
  emits: ["delete"],
  components: {
    EnvironmentSelect,
  },
  props: {
    instanceId: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      new: props.instanceId.toLowerCase() == "new",
      showDeleteModal: false,
      showPassword: false,
    });

    onMounted(() => {
      document.addEventListener("keydown", (e) => {
        if (e.code == "Escape") {
          goBack();
        }
      });
    });

    const assignInstance = (instance: Instance | NewInstance) => {
      if (!state.new) {
        state.originalInstance = instance as Instance;
      }
      // Make hard copy since we are going to make equal comparsion to determine the update button enable state.
      state.instance = JSON.parse(JSON.stringify(state.originalInstance));
    };

    const assignAdminDataSource = (dataSource: DataSource | NewDataSource) => {
      if (!state.new) {
        state.originalAdminDataSource = dataSource as DataSource;
      }
      // Make hard copy since we are going to make equal comparsion to determine the update button enable state.
      state.adminDataSource = JSON.parse(
        JSON.stringify(state.originalAdminDataSource)
      );
    };

    const updateInstance = (field: string, value: string) => {
      state.instance!.attributes[field] = value;
    };

    const updateDataSource = (field: string, value: string) => {
      state.adminDataSource!.attributes[field] = value;
    };

    const revertInstanceAndDataSource = () => {
      state.instance = JSON.parse(JSON.stringify(state.originalInstance));
      state.adminDataSource = JSON.parse(
        JSON.stringify(state.originalAdminDataSource)
      );
    };

    // [NOTE] Ternary operator doesn't trigger VS type checking, so we use a separate
    // IF block.
    if (state.new) {
      state.instance = {
        type: "instance",
        attributes: {
          name: "New Instance",
          host: "127.0.0.1",
          username: "root",
        },
      };
      state.adminDataSource = {
        type: "dataSource",
        attributes: {
          type: "ADMIN",
        },
      };
    } else {
      // Instance is already fetched remotely during routing, so we can just
      // use store.getters here.
      assignInstance(store.getters["instance/instanceById"](props.instanceId));

      // On the other hand, we need to fetch data source remotely first and
      // because the operation is async, we need to have a init object to avoid
      // adding v-if="state.adminDataSource" guard
      assignAdminDataSource({
        type: "datasource",
        attributes: {
          type: "ADMIN",
        },
      });
      store
        .dispatch(
          "datasource/fetchDataSourceListByInstanceId",
          props.instanceId
        )
        .then(() => {
          const dataSource = store.getters[
            "datasource/adminDataSourceByInstanceId"
          ](props.instanceId);
          if (dataSource) {
            assignAdminDataSource(dataSource);
          }
        })
        .catch((error) => {
          console.log(error);
        });
    }

    const valueChanged = computed(() => {
      return (
        state.new ||
        !isEqual(state.originalInstance, state.instance) ||
        !isEqual(state.originalAdminDataSource, state.adminDataSource)
      );
    });

    const goBack = () => {
      router.go(-1);
    };

    // Both doCreate and doUpdate make instance and data source create/patch in
    // seperate API. In the unlikely event, instance create/patch may succeed while
    // the corresponding data source operation failed. We consiciously make this
    // trade-off to make instance create/patch API clean without coupling data source logic.
    // The logic here to group instance/data source operation together is a shortcut
    // for the sake of UX, which shouldn't affect underlying modeling anyway.
    const doCreate = (
      newInstance: NewInstance,
      newAdminDataSource: NewDataSource
    ) => {
      store
        .dispatch("instance/createInstance", {
          newInstance,
        })
        .then((instance) => {
          store
            .dispatch("datasource/createDataSource", {
              instanceId: instance.id,
              newDataSource: newAdminDataSource,
            })
            .then((dataSource) => {
              router.push({
                name: "workspace.instance",
              });
            })
            .catch((error) => {
              console.log(error);
            });
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doUpdate = (
      updatedInstance: Instance,
      updatedAdminDataSource: DataSource
    ) => {
      store
        .dispatch("instance/patchInstanceById", {
          instanceId: props.instanceId,
          instance: updatedInstance,
        })
        .then((instance) => {
          assignInstance(instance);

          store
            .dispatch("datasource/patchDataSourceById", {
              dataSourceId: {
                id: updatedAdminDataSource.id,
                instanceId: updatedInstance.id,
              },
              dataSource: updatedAdminDataSource,
            })
            .then((dataSource) => {
              assignAdminDataSource(dataSource);
              router.push({
                name: "workspace.instance",
              });
            })
            .catch((error) => {
              console.log(error);
            });
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doDelete = () => {
      emit("delete", state.instance);
    };

    return {
      state,
      valueChanged,
      goBack,
      updateInstance,
      updateDataSource,
      revertInstanceAndDataSource,
      doCreate,
      doUpdate,
      doDelete,
      urlfy,
    };
  },
};
</script>
