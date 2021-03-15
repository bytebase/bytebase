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
  <form class="px-4 space-y-6">
    <!-- Instance Name -->
    <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
      <div class="sm:col-span-2">
        <label for="name" class="textlabel">
          Instance Name <span style="color: red">*</span>
        </label>
        <input
          required
          id="name"
          name="name"
          type="text"
          class="textfield mt-1 w-full"
          :value="state.instance.name"
          @input="updateInstance('name', $event.target.value)"
        />
      </div>

      <div class="sm:col-span-2 sm:col-start-1">
        <label for="environment" class="textlabel">
          Environment <span style="color: red">*</span>
        </label>
        <EnvironmentSelect
          id="environment"
          name="environment"
          :selectedId="state.instance.environmentId"
          @select-environment-id="
            (environmentId) => {
              updateInstance('environmentId', environmentId);
            }
          "
        />
      </div>

      <div class="sm:col-span-5">
        <label for="host" class="textlabel block">
          Host or Socket <span style="color: red">*</span>
        </label>
        <div class="mt-1">
          <input
            required
            type="text"
            id="host"
            name="host"
            placeholder="e.g. 127.0.0.1 | localhost | /tmp/mysql.sock"
            class="textfield mt-1 w-full"
            :value="state.instance.host"
            @input="updateInstance('host', $event.target.value)"
          />
        </div>
      </div>

      <div class="sm:col-span-1">
        <label for="port" class="textlabel block"> Port </label>
        <div class="mt-1">
          <input
            type="number"
            id="port"
            name="port"
            placeholder="3306"
            class="textfield mt-1 w-full"
            :value="state.instance.port"
            @input="updateInstance('port', $event.target.value)"
          />
        </div>
      </div>

      <div class="sm:col-span-6 sm:col-start-1">
        <label for="externallink" class="textlabel inline-flex">
          <span class="">External Link</span>
          <button
            class="ml-1 btn-icon"
            :disabled="state.instance.externalLink?.trim().length == 0"
            @click.prevent="
              window.open(urlfy(state.instance.externalLink), '_blank')
            "
          >
            <svg
              class="w-4 h-4"
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
          class="textfield mt-1 w-full"
          :value="state.instance.externalLink"
          @input="updateInstance('externalLink', $event.target.value)"
        />
      </div>
    </div>
    <!-- Read/Write Datasource Info -->
    <div v-if="state.new" class="py-6 border-t border-b divide-control-border">
      <div class="flex justify-between">
        <div>
          <h3 class="text-lg leading-6 font-medium text-gray-900">
            Read/Write Data Source Info
          </h3>
          <p class="mt-1 text-sm text-gray-500">
            This is the data source used by bytebase to perform DDL and
            non-SELECT DML operations. Note, bytebase does NOT need admin/SUPER
            privilege. TODO: Add grant statement.
          </p>
        </div>
      </div>
      <div class="mt-4">
        <button type="button" class="btn-normal">Test Connection</button>
      </div>
      <div class="pt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="username" class="textlabel block">
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
              class="textfield mt-1 w-full"
              :value="state.adminDataSource.username"
              @input="updateDataSource('username', $event.target.value)"
            />
          </div>
        </div>

        <div class="sm:col-span-2 sm:col-start-1">
          <label for="password" class="textlabel block"> Password </label>
          <div class="mt-1 inline-flex">
            <input
              id="password"
              name="password"
              autocomplete="off"
              :type="state.showPassword ? 'text' : 'password'"
              class="textfield mt-1 w-full"
              :value="state.adminDataSource.password"
              @input="updateDataSource('password', $event.target.value)"
            />
            <div class="ml-2 flex items-center">
              <button
                class="btn-icon"
                @click.prevent="state.showPassword = !state.showPassword"
              >
                <svg
                  v-if="state.showPassword"
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
                    d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
                  ></path>
                </svg>
                <svg
                  v-else
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
      <div v-if="state.new" class="flex justify-end">
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
      <div v-else class="flex justify-between">
        <button
          type="button"
          class="btn-danger py-2 px-4"
          @click.prevent="state.showDeleteModal = true"
        >
          Delete
        </button>
        <button
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          :disabled="!valueChanged"
          @click.prevent="doUpdate(state.instance)"
        >
          Update
        </button>
      </div>
    </div>

    <div
      v-if="!state.new"
      class="py-6 space-y-4 border-t divide-control-border"
    >
      <DataSourceTable :instance="state.instance" />
    </div>
  </form>
  <BBAlert
    v-if="state.showDeleteModal"
    :style="'CRITICAL'"
    :okText="'Delete'"
    :title="'Delete instance \'' + state.instance.name + '\' ?'"
    @ok="
      () => {
        state.showDeleteModal = false;
        doDelete();
      }
    "
    @cancel="state.showDeleteModal = false"
  >
  </BBAlert>
  <BBAlert
    v-if="state.showCancelModal"
    :style="'WARN'"
    :okText="'Discard'"
    :cancelText="'No'"
    :title="'Discard changes?'"
    @ok="
      () => {
        state.showCancelModal = false;
        goBack();
      }
    "
    @cancel="state.showCancelModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { computed, onMounted, onUnmounted, reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import { urlfy, idFromSlug } from "../utils";
import DataSourceTable from "../components/DataSourceTable.vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import { Instance, InstanceNew, DataSource, DataSourceNew } from "../types";

const INIT_DATA_SOURCE: DataSourceNew = {
  name: "Read/Write Data Source",
  type: "RW",
  username: "root",
};

interface LocalState {
  new: boolean;
  originalInstance?: Instance;
  instance?: Instance | InstanceNew;
  adminDataSource?: DataSource | DataSourceNew;
  showDeleteModal: boolean;
  showCancelModal: boolean;
  showPassword: boolean;
}

export default {
  name: "InstanceDetail",
  emits: ["delete"],
  components: {
    DataSourceTable,
    EnvironmentSelect,
  },
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      new: props.instanceSlug.toLowerCase() == "new",
      showDeleteModal: false,
      showCancelModal: false,
      showPassword: false,
    });

    const escHandler = (e: KeyboardEvent) => {
      if (
        e.code == "Escape" &&
        !state.showDeleteModal &&
        !state.showCancelModal
      ) {
        cancel();
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", escHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", escHandler);
    });

    const assignInstance = (instance: Instance | InstanceNew) => {
      if (!state.new) {
        state.originalInstance = instance as Instance;
      }
      // Make hard copy since we are going to make equal comparsion to determine the update button enable state.
      state.instance = cloneDeep(state.originalInstance);
    };

    const updateInstance = (field: string, value: string) => {
      (state.instance as any)[field] = value;
    };

    const updateDataSource = (field: string, value: string) => {
      (state.adminDataSource as any)[field] = value;
    };

    const cancel = () => {
      if (valueChanged.value) {
        state.showCancelModal = true;
      } else {
        goBack();
      }
    };

    // [NOTE] Ternary operator doesn't trigger VS type checking, so we use a separate
    // IF block.
    if (state.new) {
      state.instance = {
        name: "New Instance",
        environment: {
          id: "-1",
          name: "<<Unknown Environment",
          order: -1,
        },
        host: "127.0.0.1",
      };
      state.adminDataSource = cloneDeep(INIT_DATA_SOURCE);
    } else {
      // Instance is already fetched remotely during routing, so we can just
      // use store.getters here.
      assignInstance(
        store.getters["instance/instanceById"](idFromSlug(props.instanceSlug))
      );
    }

    const valueChanged = computed(() => {
      return state.new || !isEqual(state.originalInstance, state.instance);
    });

    const goBack = () => {
      router.go(-1);
    };

    // Both doCreate/Update/Delete make instance and data source create/patch/delete in
    // seperate API. In the unlikely event, instance operation may succeed while
    // the corresponding data source operation failed. We consiciously make this
    // trade-off to make instance create/patch API clean without coupling data source logic.
    // The logic here to group instance/data source operation together is a shortcut
    // for the sake of UX, which shouldn't affect underlying modeling anyway.
    const doCreate = (
      newInstance: InstanceNew,
      newAdminDataSource: DataSourceNew
    ) => {
      store
        .dispatch("instance/createInstance", newInstance)
        .then((instance) => {
          store
            .dispatch("dataSource/createDataSource", {
              instanceId: instance.id,
              newDataSource: newAdminDataSource,
            })
            .then((dataSource) => {
              store.dispatch("notification/pushNotification", {
                module: "bytebase",
                style: "SUCCESS",
                title: `Successfully created instance '${instance.name}'.`,
              });
              router.push({
                name: "workspace.instance",
              });

              store.dispatch("uistate/saveIntroStateByKey", {
                key: "instance.create",
                newState: true,
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

    const doUpdate = (updatedInstance: Instance) => {
      store
        .dispatch("instance/patchInstance", updatedInstance)
        .then((instance) => {
          assignInstance(instance);

          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully updated instance '${updatedInstance.name}'.`,
          });
          router.push({
            name: "workspace.instance",
          });
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doDelete = () => {
      store
        .dispatch("instance/deleteInstanceById", state.originalInstance!.id)
        .then(() => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully deleted instance '${
              state.originalInstance!.name
            }'.`,
          });
          router.push({
            name: "workspace.instance",
          });
        })
        .catch((error) => {
          console.log(error);
        });
    };

    return {
      state,
      valueChanged,
      goBack,
      updateInstance,
      updateDataSource,
      cancel,
      doCreate,
      doUpdate,
      doDelete,
      urlfy,
    };
  },
};
</script>
