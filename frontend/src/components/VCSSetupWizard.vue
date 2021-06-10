<template>
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
import {
  isValidVCSApplicationIdOrSecret,
  VCSConfig,
  VCSCreate,
} from "../types";
import { isURL } from "../utils";
import { useStore } from "vuex";

const stepList: BBStepTabItem[] = [
  { title: "Choose Git provider" },
  { title: "Fill basic info" },
  { title: "Configure settings" },
];

interface LocalState {
  config: VCSConfig;
  currentStep: number;
}

export default {
  name: "VCSSetupWizard",
  props: {},
  components: {
    VCSProviderSelectionPanel,
    VCSProviderInfoPanel,
    VCSProviderConfigPanel,
  },
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const state = reactive<LocalState>({
      config: {
        type: "GITLAB_SELF_HOST",
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
        return isURL(state.config.instanceURL);
      } else if (state.currentStep == 2) {
        return (
          isValidVCSApplicationIdOrSecret(state.config.applicationId) &&
          isValidVCSApplicationIdOrSecret(state.config.secret)
        );
      }
      return true;
    });

    const selectStep = (step: number) => {
      state.currentStep = step;
    };

    const finishSetup = () => {
      if (state.config.name == "") {
        if (state.config.type == "GITLAB_SELF_HOST") {
          state.config.name = state.config.instanceURL;
        }
      }
      const vcsCreate: VCSCreate = {
        creatorId: currentUser.value.id,
        ...state.config,
      };
      store.dispatch("vcs/createVCS", vcsCreate);
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
