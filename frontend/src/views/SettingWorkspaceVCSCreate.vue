<template>
  <div class="mt-4 space-y-4 divide-y divide-block-border">
    <div class="textinfolabel">
      Before any project can enable version control workflow, Bytebase first
      needs to integrate with the corresponding version control system (VCS) by
      registering as an OAuth application in that VCS. Below are the steps to
      configure this. You can also follow our<a
        href="https://docs.bytebase.com/use-bytebase/vcs-integration?ref=console"
        target="_blank"
        class="normal-link"
      >
        detailed guide</a
      >.
    </div>
    <div class="pt-4">
      <VCSSetupWizard />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "vue";
import VCSSetupWizard from "../components/VCSSetupWizard.vue";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default {
  name: "SettingWorkspaceVCSCreate",
  components: {
    VCSSetupWizard,
  },
  setup() {
    const state = reactive<LocalState>({});

    return {
      state,
    };
  },
};
</script>
