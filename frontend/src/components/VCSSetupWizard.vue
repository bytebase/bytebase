<template>
  <BBAttention :description="attentionText" />
  <BBStepTab
    class="mt-4"
    :stepItemList="stepList"
    :allowNext="allowNext"
    @select-step="selectStep"
    @finish="finishSetup"
    @cancel="cancelSetup"
  >
    <template v-slot:0>
      <VCSProviderSelectionPanel :config="state.config" />
    </template>
    <template v-slot:1>
      <VCSProviderInfoPanel :config="state.config" />
    </template>
    <template v-slot:2>
      <VCSProviderOAuthPanel :config="state.config" />
    </template>
  </BBStepTab>
</template>

<script lang="ts">
import { computed } from "@vue/runtime-core";
import { reactive } from "@vue/reactivity";
import { useRouter } from "vue-router";
import { BBStepTabItem } from "../bbkit/types";
import VCSProviderSelectionPanel from "./VCSProviderSelectionPanel.vue";
import VCSProviderInfoPanel from "./VCSProviderInfoPanel.vue";
import VCSProviderOAuthPanel from "./VCSProviderOAuthPanel.vue";
import {
  isValidVCSApplicationIdOrSecret,
  VCSConfig,
  VCSCreate,
} from "../types";
import { isURL } from "../utils";
import { useStore } from "vuex";

const stepList: BBStepTabItem[] = [
  { title: "Choose Git provider" },
  { title: "Basic info" },
  { title: "OAuth application info" },
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
    VCSProviderOAuthPanel,
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

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

    const attentionText = computed((): string => {
      if (state.config.type == "GITLAB_SELF_HOST") {
        return "You need to be an Admin of your chosen GitLab instance to configure this. Otherwise, you need to ask your GitLab instance Admin to register a system OAuth application for Bytebase, then provide you that Application ID and Secret to fill in Step 3.";
      }
      return "";
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

    const cancelSetup = () => {
      router.push({
        name: "setting.workspace.version-control",
      });
    };

    return {
      stepList,
      state,
      allowNext,
      attentionText,
      selectStep,
      finishSetup,
      cancelSetup,
    };
  },
};
</script>
