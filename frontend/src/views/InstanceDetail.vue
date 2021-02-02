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
  <form
    class="px-4 space-y-6 divide-y divide-control-border"
    @submit.prevent="doUpdate(state.instance)"
  >
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
        <button type="button" class="btn-normal">Test Connection</button>
      </div>
      <div class="pt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="username" class="block text-sm font-medium text-gray-700">
            Username <span style="color: red">*</span>
          </label>
          <div class="mt-1">
            <input
              required
              id="username"
              name="username"
              type="text"
              class="shadow-sm focus:ring-accent block w-full sm:text-sm border-control-border rounded-md"
              :value="state.instance.attributes.username"
              @input="updateInstance('username', $event.target.value)"
            />
          </div>
        </div>

        <div class="sm:col-span-2 sm:col-start-1">
          <label for="password" class="block text-sm font-medium text-gray-700">
            Password
          </label>
          <div class="mt-1">
            <input
              id="password"
              name="password"
              type="password"
              class="shadow-sm focus:ring-accent block w-full sm:text-sm border-control-border rounded-md"
              :value="state.instance.attributes.password"
              @input="updateInstance('password', $event.target.value)"
            />
          </div>
        </div>

        <div class="sm:col-span-2 sm:col-start-1">
          <label for="database" class="block text-sm font-medium text-gray-700">
            Database
          </label>
          <div class="mt-1">
            <input
              id="database"
              name="database"
              type="text"
              class="shadow-sm focus:ring-accent block w-full sm:text-sm border-control-border rounded-md"
              :value="state.instance.attributes.database"
              @input="updateInstance('database', $event.target.value)"
            />
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
          type="submit"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
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
            @click.prevent="revertInstance"
          >
            Revert
          </button>
          <button
            type="submit"
            class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
            :disabled="!valueChanged"
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
import { Instance, NewInstance } from "../types";

interface LocalState {
  new: boolean;
  originalInstance?: Instance;
  instance?: Instance | NewInstance;
  showDeleteModal: boolean;
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
    });

    onMounted(() => {
      document.addEventListener("keydown", (e) => {
        if (e.code == "Escape") {
          goBack();
        }
      });
    });

    const assignInstance = (instance: Instance) => {
      state.originalInstance = instance;
      // Make hard copy since we are going to make equal comparsion to determine the update button enable state.
      state.instance = JSON.parse(JSON.stringify(state.originalInstance));
    };

    const updateInstance = (field: string, value: string) => {
      state.instance!.attributes[field] = value;
    };

    const revertInstance = () => {
      state.instance = JSON.parse(JSON.stringify(state.originalInstance));
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
    } else {
      assignInstance(store.getters["instance/instanceById"](props.instanceId));
    }

    const valueChanged = computed(() => {
      return state.new || !isEqual(state.originalInstance, state.instance);
    });

    const goBack = () => {
      router.go(-1);
    };

    const doUpdate = (newInstance: Instance) => {
      store
        .dispatch("instance/patchInstanceById", {
          instanceId: props.instanceId,
          instance: newInstance,
        })
        .then((instance) => {
          assignInstance(instance);
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
      revertInstance,
      doUpdate,
      doDelete,
      urlfy,
    };
  },
};
</script>
