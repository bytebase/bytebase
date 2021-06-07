<template>
  <h1>VCS</h1>
  <BBStepTab
    :stepItemList="stepList"
    :allowNext="allowNext"
    @select-step="selectStep"
    @finish="finishSetup"
  >
    <template v-slot:0>
      <VCSProviderSelectionPanel :config="state.config" />
    </template>
    <template v-slot:1>
      <VCSProviderInfoPanel :config="state.config" />
    </template>
    <template v-slot:2>
      <VCSProviderConfigPanel :config="state.config" />
    </template>
  </BBStepTab>
</template>

<script lang="ts">
import { computed } from "@vue/runtime-core";
import { reactive } from "@vue/reactivity";
import { BBStepTabItem } from "../bbkit/types";
import VCSProviderSelectionPanel from "./VCSProviderSelectionPanel.vue";
import VCSProviderInfoPanel from "./VCSProviderInfoPanel.vue";
import VCSProviderConfigPanel from "./VCSProviderConfigPanel.vue";
import { isValidApplicationIdOrSecret, VCSConfig } from "../types";
import { isUrl } from "../utils";

const stepList: BBStepTabItem[] = [
  { title: "Choose Git provider" },
  { title: "Fill provider basic info" },
  { title: "Configure settings" },
];

interface LocalState {
  config: VCSConfig;
  currentStep: number;
}

export default {
  name: "SettingWorkspaceVCS",
  props: {},
  components: {
    VCSProviderSelectionPanel,
    VCSProviderInfoPanel,
    VCSProviderConfigPanel,
  },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      config: {
        vcsType: "GITLAB_SELF_HOST",
        name: "",
        instanceURL: "",
        applicationId: "",
        secret: "",
      },
      currentStep: 0,
    });

    const allowNext = computed((): boolean => {
      if (state.currentStep == 0) {
        return true;
      } else if (state.currentStep == 1) {
        return isUrl(state.config.instanceURL);
      } else if (state.currentStep == 2) {
        return (
          isValidApplicationIdOrSecret(state.config.applicationId) &&
          isValidApplicationIdOrSecret(state.config.secret)
        );
      }
      return true;
    });

    const selectStep = (step: number) => {
      state.currentStep = step;
    };

    const finishSetup = () => {
      console.log("finish", state.config);
    };

    return {
      stepList,
      state,
      allowNext,
      selectStep,
      finishSetup,
    };
  },
};
</script>
