<template>
  <div
    class="align-middle inline-block min-w-full border-b border-block-border"
  >
    <table class="min-w-full">
      <thead>
        <tr class="border-t border-block-border">
          <th
            class="px-6 py-3 border-b border-block-border bg-gray-50 text-left text-xs leading-4 font-medium text-gray-500 uppercase tracking-wider"
          >
            <span class="lg:pl-2">Project</span>
          </th>
          <th
            class="hidden md:table-cell px-6 py-3 border-b border-block-border bg-gray-50 text-left text-xs leading-4 font-medium text-gray-500 uppercase tracking-wider"
          >
            Slug
          </th>
          <th
            class="hidden md:table-cell px-6 py-3 border-b border-block-border bg-gray-50 text-right text-xs leading-4 font-medium text-gray-500 uppercase tracking-wider"
          >
            Last updated
          </th>
          <th
            class="pr-6 py-3 border-b border-block-border bg-gray-50 text-right text-xs leading-4 font-medium text-gray-500 uppercase tracking-wider"
          ></th>
        </tr>
      </thead>
      <tbody class="bg-normal divide-y divide-gray-100">
        <tr v-for="project in projectList" :key="project.id">
          <td
            class="px-6 py-3 max-w-0 w-full whitespace-nowrap text-sm leading-5 font-medium text-gray-900"
          >
            <div class="flex items-center space-x-3 lg:pl-2">
              <router-link
                :to="`/${project.attributes.namespace}/${project.attributes.slug}`"
                class="truncate hover:text-gray-600"
              >
                {{ project.attributes.name }}
              </router-link>
            </div>
          </td>
          <td
            class="hidden md:table-cell px-6 py-3 whitespace-nowrap text-sm leading-5 text-gray-900 text-left"
          >
            {{ project.attributes.slug }}
          </td>
          <td
            class="hidden md:table-cell px-6 py-3 whitespace-nowrap text-sm leading-5 text-gray-500 text-right"
          >
            March 17, 2020
          </td>
          <td class="pr-6">
            <div class="relative flex justify-end items-center">
              <ProjectTableActionButton />
            </div>
          </td>
        </tr>

        <!-- More project rows... -->
      </tbody>
    </table>
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useStore } from "vuex";

import ProjectTableActionButton from "./ProjectTableActionButton.vue";
import { User } from "../types";

export default {
  name: "ProjectTable",
  components: {
    ProjectTableActionButton,
  },
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser: User = computed(() =>
      store.getters["auth/currentUser"]()
    ).value;

    const projectList = computed(() =>
      store.getters["project/projectListByUser"](currentUser.id)
    );

    return {
      projectList,
    };
  },
};
</script>
