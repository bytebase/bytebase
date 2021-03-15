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
              <dl class="flex flex-col mt-2 sm:flex-row sm:flex-wrap">
                <dt class="sr-only">RoleType</dt>
                <dd
                  class="flex items-center text-sm text-control-light font-medium sm:mr-4"
                >
                  <!-- Heroicon name: solid/pencil -->
                  <svg
                    v-if="dataSource.type == 'RW'"
                    class="mr-1.5 w-5 h-5"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
                    ></path>
                  </svg>
                  <!-- Heroicon name: solid/pencil -->
                  <svg
                    v-if="dataSource.type == 'RO'"
                    class="mr-1.5 w-5 h-5"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path d="M10 12a2 2 0 100-4 2 2 0 000 4z"></path>
                    <path
                      fill-rule="evenodd"
                      d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z"
                      clip-rule="evenodd"
                    ></path>
                  </svg>
                  <span v-data-source-type class="text-control">{{
                    dataSource.type
                  }}</span>
                </dd>
                <dt class="sr-only">Instance</dt>
                <dd
                  class="flex items-center text-sm text-control-light font-medium sm:mr-4"
                >
                  <!-- Heroicon name: solid/database -->
                  <svg
                    class="mr-1.5 w-5 h-5"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      d="M3 12v3c0 1.657 3.134 3 7 3s7-1.343 7-3v-3c0 1.657-3.134 3-7 3s-7-1.343-7-3z"
                    ></path>
                    <path
                      d="M3 7v3c0 1.657 3.134 3 7 3s7-1.343 7-3V7c0 1.657-3.134 3-7 3S3 8.657 3 7z"
                    ></path>
                    <path
                      d="M17 5c0 1.657-3.134 3-7 3S3 6.657 3 5s3.134-3 7-3 7 1.343 7 3z"
                    ></path>
                  </svg>
                  <router-link
                    :to="`/instance/${instanceSlug}`"
                    class="normal-link"
                  >
                    {{ instance.name }} -
                    {{ databaseName }}
                  </router-link>
                </dd>
                <dt class="sr-only">Environment</dt>
                <dd
                  class="flex items-center text-sm text-control-light font-medium"
                >
                  <!-- Heroicon name: solid/globle-alt -->
                  <svg
                    class="mr-1.5 w-5 h-5"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      fill-rule="evenodd"
                      d="M4.083 9h1.946c.089-1.546.383-2.97.837-4.118A6.004 6.004 0 004.083 9zM10 2a8 8 0 100 16 8 8 0 000-16zm0 2c-.076 0-.232.032-.465.262-.238.234-.497.623-.737 1.182-.389.907-.673 2.142-.766 3.556h3.936c-.093-1.414-.377-2.649-.766-3.556-.24-.56-.5-.948-.737-1.182C10.232 4.032 10.076 4 10 4zm3.971 5c-.089-1.546-.383-2.97-.837-4.118A6.004 6.004 0 0115.917 9h-1.946zm-2.003 2H8.032c.093 1.414.377 2.649.766 3.556.24.56.5.948.737 1.182.233.23.389.262.465.262.076 0 .232-.032.465-.262.238-.234.498-.623.737-1.182.389-.907.673-2.142.766-3.556zm1.166 4.118c.454-1.147.748-2.572.837-4.118h1.946a6.004 6.004 0 01-2.783 4.118zm-6.268 0C6.412 13.97 6.118 12.546 6.03 11H4.083a6.004 6.004 0 002.783 4.118z"
                      clip-rule="evenodd"
                    ></path>
                  </svg>
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
          <button
            v-else
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
        </div>
      </div>

      <form class="mt-6">
        <div class="max-w-6xl mx-auto px-4">
          <!-- Description list -->
          <div class="max-w-5xl mx-auto px-4 mb-6">
            <dl class="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-2">
              <div class="sm:col-span-1">
                <dt class="text-sm font-medium text-control-light">
                  Username<span v-if="state.editing" class="text-red-600"
                    >*</span
                  >
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
                  <template v-else>
                    {{ dataSource.username }}
                  </template>
                </dd>
              </div>

              <div class="sm:col-span-1">
                <div class="flex items-center space-x-1">
                  <dt class="text-sm font-medium text-control-light">
                    Password
                  </dt>
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
                  <template v-else-if="state.showPassword">
                    {{ dataSource.password }}
                  </template>
                  <template v-else> ****** </template>
                </dd>
              </div>
            </dl>
          </div>

          <DataSourceMemberTable
            :instanceId="instance.id"
            :dataSourceId="dataSource.id"
          />
        </div>
      </form>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, nextTick, reactive, ref } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import DataSourceMemberTable from "../components/DataSourceMemberTable.vue";
import { idFromSlug } from "../utils";
import { DataSource } from "../types";

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
    const instanceId = idFromSlug(props.instanceSlug);
    const dataSourceId = idFromSlug(props.dataSourceSlug);

    const state = reactive<LocalState>({
      editing: false,
      showPassword: false,
    });

    const dataSource = computed(() => {
      return store.getters["dataSource/dataSourceById"](
        dataSourceId,
        instanceId
      );
    });

    const instance = computed(() => {
      return store.getters["instance/instanceById"](instanceId);
    });

    const databaseName = computed(() => {
      if (dataSource.value.databaseId) {
        const database = store.getters["database/databaseById"](
          dataSource.value.databaseId,
          instanceId
        );
        return database.name;
      }
      return "* (All databases)";
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

    return {
      editNameTextField,
      state,
      dataSource,
      instance,
      databaseName,
      allowSave,
      editDataSource,
      cancelEdit,
      saveEdit,
    };
  },
};
</script>
