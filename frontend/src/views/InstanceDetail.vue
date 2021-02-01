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
    class="px-4 space-y-6 divide-y divide-gray-200"
    @submit.prevent="doUpdate(state.instance)"
  >
    <div class="pt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
      <div class="sm:col-span-2">
        <label for="name" class="text-lg leading-6 font-medium text-gray-900">
          Instance Name <span style="color: red">*</span>
        </label>
        <input
          required
          id="name"
          name="name"
          type="text"
          class="shadow-sm mt-4 focus:ring-accent block w-full sm:text-sm border-control-border rounded-md"
          :value="state.instance.attributes.name"
          @input="updateInstance('name', $event.target.value)"
        />
      </div>
    </div>
    <div class="pt-6">
      <h3 class="text-lg leading-6 font-medium text-gray-900">
        Connection Info
      </h3>
      <p class="mt-1 text-sm text-gray-500">
        Provide the info to connect to the database.
      </p>
      <button type="button" class="btn-normal mt-4">Test Connection</button>
    </div>
    <div class="pt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
      <div class="sm:col-span-3">
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

      <div class="sm:col-span-3">
        <label for="port" class="block text-sm font-medium text-gray-700">
          Port (Only applicable when connecting via Host)
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

      <div class="sm:col-span-2">
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
  components: {},
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
      router.push(store.getters["router/backPath"]());
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
      doUpdate,
      doDelete,
    };
  },
};
</script>
