<template>
  <RepositorySetupWizard />
</template>

<script lang="ts">
import RepositorySetupWizard from "./RepositorySetupWizard.vue";

export default {
  name: "ProjectVCSPanel",
  components: {
    RepositorySetupWizard,
  },
  props: {},
  async setup(props) {},
};
</script>
