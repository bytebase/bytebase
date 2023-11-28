<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="textinfolabel">
      <i18n-t keypath="gitops.setting.add-git-provider.description">
        <template #guide>
          <a
            href="https://bytebase.com/docs/vcs-integration/overview?source=console"
            target="_blank"
            class="normal-link"
          >
            {{ $t("common.detailed-guide") }}</a
          >
        </template>
      </i18n-t>
    </div>
    <div class="pt-4">
      <VCSSetupWizard />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "vue";
import VCSSetupWizard from "@/components/VCS/VCSSetupWizard.vue";

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
