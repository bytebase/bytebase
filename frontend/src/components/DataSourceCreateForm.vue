<template>
  <form
    class="space-y-6 divide-y divide-block-border"
    @submit.prevent="$emit('create', state.dataSource)"
  >
    <div class="space-y-4">
      <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-3">
        <div class="sm:col-span-2">
          <label for="name" class="textlabel">
            Name <span class="text-red-600">*</span>
          </label>
          <input
            id="name"
            required
            name="name"
            type="text"
            class="textfield mt-1 w-full"
            :value="state.dataSource.name"
            @input="updateDataSource('name', $event.target.value)"
          />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-3 items-center">
        <div class="sm:col-span-2">
          <label for="database" class="textlabel">
            Database <span class="text-red-600">*</span>
          </label>
          <div v-if="database" class="flex flex-row justify-between space-x-2">
            {{ database.name }}
            <BBSwitch
              :label="'Read-only'"
              :value="state.dataSource.type == 'RO'"
              @toggle="
                (on) => {
                  updateDataSource('type', on ? 'RO' : 'RW');
                }
              "
            />
          </div>
          <div v-else class="flex flex-row justify-between space-x-2">
            <!-- eslint-disable vue/attribute-hyphenation -->
            <DatabaseSelect
              class="mt-1 w-full"
              :selectedId="state.dataSource.databaseId"
              :mode="'INSTANCE'"
              :instanceId="instanceId"
              @select-database-id="
                (databaseId) => {
                  updateDataSource('databaseId', databaseId);
                }
              "
            />
            <BBSwitch
              :label="'Read-only'"
              :value="state.dataSource.type == 'RO'"
              @toggle="
                (on) => {
                  updateDataSource('type', on ? 'RO' : 'RW');
                }
              "
            />
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-3">
        <div class="sm:col-span-2">
          <label for="username" class="textlabel"> Username </label>
          <input
            id="username"
            name="username"
            type="text"
            class="textfield mt-1 w-full"
            :value="state.dataSource.username"
            @input="updateDataSource('username', $event.target.value)"
          />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-3">
        <div class="sm:col-span-2">
          <div class="flex flex-row items-center space-x-1">
            <label for="username" class="textlabel"> Password </label>
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
          <input
            id="password"
            name="password"
            autocomplete="off"
            :type="state.showPassword ? 'text' : 'password'"
            class="textfield mt-1 w-full"
            :value="state.dataSource.password"
            @input="updateDataSource('password', $event.target.value)"
          />
        </div>
      </div>
    </div>
    <!-- Create button group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="$emit('cancel')"
      >
        Cancel
      </button>
      <button
        type="submit"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
      >
        Create
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, PropType, reactive } from "vue";
import DatabaseSelect from "./DatabaseSelect.vue";
import { DataSourceCreate, Database, UNKNOWN_ID } from "../types";

interface LocalState {
  dataSource: DataSourceCreate;
  showPassword: boolean;
}

export default {
  name: "DataSourceCreateForm",
  components: { DatabaseSelect },
  props: {
    instanceId: {
      required: true,
      type: Number,
    },
    // If database is specified, then we just show that database instead of the database select
    database: {
      type: Object as PropType<Database>,
    },
  },
  emits: ["create", "cancel"],
  setup(props) {
    // const store = useStore();

    // const currentUser: ComputedRef<Principal> = computed(() =>
    //   store.getters["auth/currentUser"]()
    // );

    const state = reactive<LocalState>({
      dataSource: {
        name: "New data source",
        type: "RO",
        databaseId: props.database ? props.database.id : UNKNOWN_ID,
        instanceId: props.instanceId,
        memberList: [],
      },
      showPassword: false,
    });

    const allowCreate = computed(() => {
      return state.dataSource.name;
    });

    const updateDataSource = (field: string, value: string) => {
      (state.dataSource as any)[field] = value;
    };

    return {
      state,
      allowCreate,
      updateDataSource,
    };
  },
};
</script>
