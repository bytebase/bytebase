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
                <h1 class="text-xl font-bold leading-7 text-main truncate">
                  {{ dataSource.name }} - {{ dataSource.type }}
                </h1>
              </div>
              <dl class="flex flex-col mt-2 sm:flex-row sm:flex-wrap">
                <dt class="sr-only">Instance</dt>
                <dd
                  class="flex items-center text-sm text-control-light font-medium sm:mr-6"
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
                  class="flex items-center text-sm text-control-light font-medium sm:mr-6"
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
          <button type="button" class="btn-normal" @click.prevent="editUser">
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

      <div class="mt-6">
        <div class="max-w-6xl mx-auto px-4">
          <!-- Description list -->
          <div class="max-w-5xl mx-auto px-4 mb-6">
            <dl class="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-2">
              <div class="sm:col-span-1">
                <dt class="text-sm font-medium text-control-light">Username</dt>
                <dd class="mt-1 text-sm text-main">
                  {{ dataSource.username }}
                </dd>
              </div>

              <div class="sm:col-span-1">
                <dt class="text-sm font-medium text-control-light">Password</dt>
                <dd class="mt-1 text-sm text-main">
                  {{ dataSource.password }}
                </dd>
              </div>
            </dl>
          </div>

          <DataSourceMemberTable
            :instanceId="instance.id"
            :dataSourceId="dataSource.id"
          />
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useStore } from "vuex";
import DataSourceMemberTable from "../components/DataSourceMemberTable.vue";
import { idFromSlug } from "../utils";

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
    const store = useStore();
    const instanceId = idFromSlug(props.instanceSlug);
    const dataSourceId = idFromSlug(props.dataSourceSlug);

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

    return {
      dataSource,
      instance,
      databaseName,
    };
  },
};
</script>
