<template>
  <div class="mt-4 space-y-4 divide-y divide-block-border">
    <div class="flex items-center justify-begin">
      <div class="text-sm text-control">
        To enable GitOps workflow where the database migration scripts are
        stored in the Git repository and automatically applies DDL changes after
        scripts being modified in the repository, Bytebase needs to access the
        corresponding version control system (VCS) instance. Below are the steps
        to configure OAuth with your VCS to obtain that access.
      </div>
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
