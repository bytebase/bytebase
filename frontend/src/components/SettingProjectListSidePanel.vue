<template>
  <div class="mt-1 space-y-1" role="group" aria-labelledby="projects-headline">
    <template v-for="project in projectList" :key="project.id">
      <router-link
        :to="`/${project.attributes.namespace}/${project.attributes.slug}/setting`"
        class="sidebar-link group w-full flex items-center pl-11 pr-2 py-2"
      >
        {{ project.attributes.name }}
      </router-link>
    </template>
  </div>
</template>

<script lang="ts">
import { watchEffect, computed } from "vue";
import { useStore } from "vuex";
import { User } from "../types";

export default {
  name: "SettingProjectListSidePanel",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser: User = computed(() =>
      store.getters["auth/currentUser"]()
    ).value;

    const prepareProjectList = () => {
      store
        .dispatch("project/fetchProjectListForUser", currentUser.id)
        .catch((error) => {
          console.log(error);
        });
    };

    const projectList = computed(() =>
      store.getters["project/projectListByUser"](currentUser.id)
    );

    watchEffect(prepareProjectList);

    return {
      projectList,
    };
  },
};
</script>
