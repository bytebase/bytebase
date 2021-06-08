<template>
  <template v-if="vcsList.length > 0">
    <template v-for="(vcs, index) in vcsList" :key="index">
      <VCSCard :vcs="vcs" />
    </template>
  </template>
  <template v-else>
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
</template>

<script lang="ts">
import { computed, watchEffect } from "@vue/runtime-core";
import { reactive } from "@vue/reactivity";
import { BBStepTabItem } from "../bbkit/types";
import VCSProviderSelectionPanel from "./VCSProviderSelectionPanel.vue";
import VCSProviderInfoPanel from "./VCSProviderInfoPanel.vue";
import VCSProviderConfigPanel from "./VCSProviderConfigPanel.vue";
import VCSCard from "../components/VCSCard.vue";
import { isValidApplicationIdOrSecret, VCSConfig, VCSCreate } from "../types";
import { isURL } from "../utils";
import { useStore } from "vuex";

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
    VCSCard,
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

    const prepareVCSList = () => {
      store.dispatch("vcs/fetchVCSList");
    };

    watchEffect(prepareVCSList);

    const vcsList = computed(() => {
      return store.getters["vcs/vcsList"]();
    });

    const allowNext = computed((): boolean => {
      if (state.currentStep == 0) {
        return true;
      } else if (state.currentStep == 1) {
        return isURL(state.config.instanceURL);
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
      const vcsCreate: VCSCreate = {
        creatorId: currentUser.value.id,
        ...state.config,
      };
      store.dispatch("vcs/createVCS", vcsCreate);
    };

    return {
      stepList,
      state,
      vcsList,
      allowNext,
      selectStep,
      finishSetup,
    };
  },
};
</script>
