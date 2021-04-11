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
  <form class="space-y-6 divide-y divide-block-border">
    <div class="">
      <!-- Instance Name -->
      <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-3">
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="environment" class="textlabel">
            Environment <span v-if="create" style="color: red">*</span>
          </label>
          <!-- Disallow changing environment after creation. This is to take the conservative approach to limit capability -->
          <EnvironmentSelect
            class="mt-1"
            id="environment"
            name="environment"
            :disabled="!create"
            :selectedId="
              create
                ? state.instance.environmentId
                : state.instance.environment.id
            "
            @select-environment-id="
              (environmentId) => {
                if (create) {
                  state.instance.environmentId = environmentId;
                } else {
                  updateInstance('environmentId', environmentId);
                }
              }
            "
          />
        </div>

        <div class="sm:col-span-2 sm:col-start-1">
          <label for="name" class="textlabel">
            Instance Name <span style="color: red">*</span>
          </label>
          <input
            required
            id="name"
            name="name"
            type="text"
            class="textfield mt-1 w-full"
            :disabled="!allowEdit"
            :value="state.instance.name"
            @input="updateInstance('name', $event.target.value)"
          />
        </div>

        <div class="sm:col-span-2 sm:col-start-1">
          <label for="host" class="textlabel block">
            Host or Socket <span style="color: red">*</span>
          </label>
          <input
            required
            type="text"
            id="host"
            name="host"
            placeholder="e.g. 127.0.0.1 | localhost | /tmp/mysql.sock"
            class="textfield mt-1 w-full"
            :disabled="!allowEdit"
            :value="state.instance.host"
            @input="updateInstance('host', $event.target.value)"
          />
        </div>

        <div class="sm:col-span-1">
          <label for="port" class="textlabel block"> Port </label>
          <input
            type="number"
            id="port"
            name="port"
            placeholder="3306"
            class="textfield mt-1 w-full"
            :disabled="!allowEdit"
            :value="state.instance.port"
            @input="updateInstance('port', $event.target.value)"
          />
        </div>

        <div class="sm:col-span-3 sm:col-start-1">
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
            :disabled="!allowEdit"
            :value="state.instance.externalLink"
            @input="updateInstance('externalLink', $event.target.value)"
          />
        </div>
      </div>
      <!-- Read/Write Datasource Info -->
      <div v-if="create" class="pt-4">
        <div class="flex justify-between">
          <div>
            <h3 class="text-lg leading-6 font-medium text-gray-900">
              Read/Write Data Source Info
            </h3>
            <p class="mt-1 text-sm text-gray-500 max-w-xl">
              This is the data source used by bytebase to perform DDL and DML
              operations. Note, bytebase does NOT need admin/SUPER privilege.
              TODO: Add grant statement.
            </p>
          </div>
        </div>
        <div class="pt-4">
          <button type="button" class="btn-normal">Test Connection</button>
        </div>
        <div class="pt-4 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-3">
          <div class="sm:col-span-1 sm:col-start-1">
            <label for="username" class="textlabel block"> Username </label>
            <!-- For mysql, username can be empty indicating anonymous user. 
            But it's a very bad practice to use anonymous user for admin operation,
            thus we make it REQUIRED here. -->
            <input
              id="username"
              name="username"
              type="text"
              class="textfield mt-1 w-full"
              :value="state.username"
              @input="state.username = $event.target.value"
            />
          </div>

          <div class="sm:col-span-1 sm:col-start-1">
            <div class="flex flex-row items-center">
              <label for="password" class="textlabel block"> Password </label>
              <div class="ml-1 flex items-center">
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
            <input
              id="password"
              name="password"
              autocomplete="off"
              :type="state.showPassword ? 'text' : 'password'"
              class="textfield mt-1 w-full"
              :value="state.password"
              @input="state.password = $event.target.value"
            />
          </div>
        </div>
      </div>
    </div>
    <!-- Action Button Group -->
    <div class="pt-4">
      <!-- Create button group -->
      <div v-if="create" class="flex justify-end">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="cancel"
        >
          Cancel
        </button>
        <button
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          :disabled="!allowCreate"
          @click.prevent="
            doCreate(state.instance, state.username, state.password)
          "
        >
          Create
        </button>
      </div>
      <!-- Update button group -->
      <div v-else class="flex justify-between">
        <template v-if="state.instance.rowStatus == 'NORMAL'">
          <BBButtonConfirm
            :style="'ARCHIVE'"
            :buttonText="'Archive this instance'"
            :okText="'Archive'"
            :requireConfirm="true"
            :confirmTitle="`Archive instance '${state.instance.name}'?`"
            :confirmDescription="'Archived instsance will not be shown on the normal interface. You can still restore later from the Archive page.'"
            @confirm="doArchive"
          />
        </template>
        <template v-else-if="state.instance.rowStatus == 'ARCHIVED'">
          <BBButtonConfirm
            :style="'RESTORE'"
            :buttonText="'Restore this instance'"
            :okText="'Restore'"
            :requireConfirm="true"
            :confirmTitle="`Restore instance '${state.instance.name}' to normal state?`"
            :confirmDescription="''"
            @confirm="doRestore"
          />
        </template>
        <button
          v-if="allowEdit"
          type="button"
          class="btn-normal ml-3 inline-flex justify-center py-2 px-4"
          :disabled="!valueChanged"
          @click.prevent="doUpdate"
        >
          Update
        </button>
      </div>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, reactive, PropType, ComputedRef } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import { instanceSlug } from "../utils";
import {
  Instance,
  InstanceNew,
  DataSourceNew,
  UNKNOWN_ID,
  ALL_DATABASE_NAME,
  User,
  InstancePatch,
} from "../types";

interface LocalState {
  originalInstance?: Instance;
  instance: Instance | InstanceNew;
  username?: string;
  password?: string;
  showPassword: Boolean;
}

export default {
  name: "DataSourceNewForm",
  emits: ["dismiss"],
  props: {
    create: {
      default: "false",
      type: Boolean,
    },
    instance: {
      // Can be false when create is true
      required: false,
      type: Object as PropType<Instance>,
    },
  },
  components: { EnvironmentSelect },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const currentUser: ComputedRef<User> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    const state = reactive<LocalState>({
      originalInstance: props.instance,
      // Make hard copy since we are going to make equal comparsion to determine the update button enable state.
      instance: props.instance
        ? cloneDeep(props.instance)
        : {
            environmentId: UNKNOWN_ID,
            name: "New Instance",
            host: "127.0.0.1",
          },
      username: "root",
      showPassword: false,
    });

    const allowCreate = computed(() => {
      return state.instance.name && state.instance.host;
    });

    const allowEdit = computed(() => {
      return props.create || (state.instance as Instance).rowStatus == "NORMAL";
    });

    const valueChanged = computed(() => {
      return !isEqual(state.instance, state.originalInstance);
    });

    const updateInstance = (field: string, value: string) => {
      (state.instance as any)[field] = value;
    };

    const cancel = () => {
      emit("dismiss");
    };

    // doCreate make instance, database and data source creation in seperate API.
    // In the unlikely event, instance operation may succeed while the corresponding
    // database/datasource operation failed. We consiciously make this trade-off to make
    // instance create API clean without coupling database, data source logic.
    // The logic here to group instance, database and data source operation together is
    // for providing better UX, which shouldn't affect underlying modeling anyway.
    const doCreate = async (
      newInstance: InstanceNew,
      username?: string,
      password?: string
    ) => {
      const createdInstance = await store.dispatch(
        "instance/createInstance",
        newInstance
      );

      // Create the database representing all databases(*)
      const createdDatabase = await store.dispatch("database/createDatabase", {
        name: ALL_DATABASE_NAME,
        instanceId: createdInstance.id,
        ownerId: currentUser.value.id,
        creatorId: currentUser.value.id,
      });

      const adminDataSource: DataSourceNew = {
        name: "Admin data source",
        databaseId: createdDatabase.id,
        instanceId: createdInstance.id,
        memberList: [],
        type: "RW",
        username,
        password,
      };

      store
        .dispatch("dataSource/createDataSource", adminDataSource)
        .then(() => {
          emit("dismiss");

          router.push(`/instance/${instanceSlug(createdInstance)}`);

          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully created instance '${createdInstance.name}'.`,
          });

          store.dispatch("uistate/saveIntroStateByKey", {
            key: "instance.create",
            newState: true,
          });
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doUpdate = () => {
      const patchedInstance: InstancePatch = {};
      if (state.instance.name != state.originalInstance!.name) {
        patchedInstance.name = state.instance.name;
      }
      if (state.instance.externalLink != state.originalInstance!.externalLink) {
        patchedInstance.externalLink = state.instance.externalLink;
      }
      if (state.instance.host != state.originalInstance!.host) {
        patchedInstance.host = state.instance.host;
      }
      if (state.instance.port != state.originalInstance!.port) {
        patchedInstance.port = state.instance.port;
      }
      store
        .dispatch("instance/patchInstance", {
          instanceId: (state.instance as Instance).id,
          instancePatch: patchedInstance,
        })
        .then((instance) => {
          state.originalInstance = instance;
          // Make hard copy since we are going to make equal comparsion to determine the update button enable state.
          state.instance = cloneDeep(state.originalInstance!);

          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully updated instance '${instance.name}'.`,
          });
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doArchive = () => {
      store
        .dispatch("instance/patchInstance", {
          instanceId: (state.instance as Instance).id,
          instancePatch: {
            rowStatus: "ARCHIVED",
          },
        })
        .then((instance) => {
          state.originalInstance = instance;
          // Make hard copy since we are going to make equal comparsion to determine the update button enable state.
          state.instance = cloneDeep(state.originalInstance!);

          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully archived instance '${instance.name}'.`,
          });
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const doRestore = () => {
      store
        .dispatch("instance/patchInstance", {
          instanceId: (state.instance as Instance).id,
          instancePatch: {
            rowStatus: "NORMAL",
          },
        })
        .then((instance) => {
          state.originalInstance = instance;
          // Make hard copy since we are going to make equal comparsion to determine the update button enable state.
          state.instance = cloneDeep(state.originalInstance!);

          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully restored instance '${instance.name}'.`,
          });
        })
        .catch((error) => {
          console.log(error);
        });
    };

    return {
      state,
      allowCreate,
      allowEdit,
      valueChanged,
      updateInstance,
      cancel,
      doCreate,
      doUpdate,
      doArchive,
      doRestore,
    };
  },
};
</script>
