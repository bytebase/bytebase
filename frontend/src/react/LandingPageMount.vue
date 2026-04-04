<template>
  <ReactPageMount page="LandingPage" v-bind="pageProps" />
  <ProjectSwitchModal
    :show="showProjectModal"
    @dismiss="showProjectModal = false"
  />
</template>

<script lang="ts" setup>
import { ref } from "vue";
import ProjectSwitchModal from "@/components/Project/ProjectSwitch/ProjectSwitchModal.vue";
import ReactPageMount from "./ReactPageMount.vue";

const showProjectModal = ref(false);

const pageProps = {
  onOpenProjectSwitch: () => {
    showProjectModal.value = true;
  },
};
</script>
