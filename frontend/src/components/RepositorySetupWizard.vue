<template>
  <BBStepTab
    :stepItemList="stepList"
    @select-step="selectStep"
    @finish="finishSetup"
  >
    <template v-slot:0="{ next }">
      <RepositoryVCSPanel :config="state.config" @next="next()" />
    </template>
    <template v-slot:1="{ next }">
      <RepositorySelectionPanel :config="state.config" @next="next()" />
    </template>
    <template v-slot:2>
      <RepositoryConfigPanel :config="state.config" />
    </template>
  </BBStepTab>
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import { BBStepTabItem } from "../bbkit/types";
import { useStore } from "vuex";
import RepositoryVCSPanel from "./RepositoryVCSPanel.vue";
import RepositorySelectionPanel from "./RepositorySelectionPanel.vue";
import RepositoryConfigPanel from "./RepositoryConfigPanel.vue";
import { ProjectRepoConfig, Repository, unknown, VCS } from "../types";

interface LocalState {
  config: ProjectRepoConfig;
}

const stepList: BBStepTabItem[] = [
  { title: "Choose Git provider", hideNext: true },
  { title: "Select repository", hideNext: true },
  { title: "Configure deploy" },
];

export default {
  name: "RepositorySetupWizard",
  components: {
    RepositoryVCSPanel,
    RepositorySelectionPanel,
    RepositoryConfigPanel,
  },
  setup(props, ctx) {
    const store = useStore();
    const state = reactive<LocalState>({
      config: {
        vcs: unknown("VCS") as VCS,
        code: "",
        accessToken: "",
        repository: unknown("REPOSITORY") as Repository,
        baseDirectory: "",
        branchFilter: "",
      },
    });

    const selectStep = (step: number) => {
      console.log("select step", step);
    };

    const finishSetup = () => {
      console.log("finish", state.config);
      store
        .dispatch("gitlab/createWebhook", {
          vcs: state.config.vcs,
          projectId: state.config.repository.externalId,
          branchFilter: state.config.branchFilter,
          token: state.config.accessToken,
        })
        .then((createdHook) => {});
    };

    return {
      state,
      stepList,
      selectStep,
      finishSetup,
    };
  },
};
</script>
