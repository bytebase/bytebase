<template>
  <h3
    class="px-3 text-xs leading-4 font-semibold text-control-light uppercase tracking-wider"
    id="projects-headline"
  >
    Projects
  </h3>
  <ProjectSidePanel
    v-for="item in projectList"
    :key="item.id"
    :project="item"
  />
</template>

<script lang="ts">
import { watchEffect, computed } from "vue";
import { useStore } from "vuex";
import ProjectSidePanel from "./ProjectSidePanel.vue";
import { User } from "../types";

export default {
  name: "ProjectListSidePanel",
  props: {},
  components: {
    ProjectSidePanel,
  },
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
