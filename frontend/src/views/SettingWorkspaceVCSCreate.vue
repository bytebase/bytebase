<template>
  <div class="mt-4 space-y-4 divide-y divide-block-border">
    <div class="textinfolabel">
      Before any project can enable version control workflow to automatically
      apply DDL on database migration scripts changes in the Git repository,
      Bytebase first needs to integrate with the corresponding version control
      system (VCS) by registering as an OAuth application in that VCS. Below are
      the steps to configure this.
    </div>
    <div class="pt-4">
      <VCSSetupWizard />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import VCSSetupWizard from "../components/VCSSetupWizard.vue";

interface LocalState {}

export default {
  name: "SettingWorkspaceVCSCreate",
  props: {},
  components: {
    VCSSetupWizard,
  },
  setup(props, ctx) {
    const state = reactive<LocalState>({});

    return {
      state,
    };
  },
};
</script>
