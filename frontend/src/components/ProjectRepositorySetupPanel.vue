<template>
  <BBStepTab
    :stepItemList="stepList"
    @select-step="selectStep"
    @finish="finishSetup"
  >
    <template v-slot:0="{ next }">
      <RepositoryVCSPanel
        @select-vcs="
          (vcs) => {
            next();
            selectVCS(vcs);
          }
        "
      />
    </template>
    <template v-slot:1> select repo </template>
    <template v-slot:2> configure deploy </template>
  </BBStepTab>
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import { BBStepTabItem } from "../bbkit/types";
import { useStore } from "vuex";
import { computed, watchEffect } from "@vue/runtime-core";
import RepositoryVCSPanel from "./RepositoryVCSPanel.vue";
import { ProjectRepoConfig, UNKNOWN_ID, VCS } from "../types";

interface LocalState {
  config: ProjectRepoConfig;
}

const stepList: BBStepTabItem[] = [
  { title: "Choose Git provider", hideNext: true },
  { title: "Select repository" },
  { title: "Configure deploy" },
];

export default {
  name: "ProjectRepositorySetupPanel",
  components: {
    RepositoryVCSPanel,
  },
  setup(props, ctx) {
    const store = useStore();
    const state = reactive<LocalState>({
      config: {
        vcsId: UNKNOWN_ID,
      },
    });

    const prepareVCSList = () => {
      store.dispatch("vcs/fetchVCSList");
    };

    watchEffect(prepareVCSList);

    const vcsList = computed(() => {
      return store.getters["vcs/vcsList"]();
    });

    const selectVCS = (vcs: VCS) => {
      state.config.vcsId = vcs.id;
    };

    const selectStep = (step: number) => {
      console.log("select step", step);
    };

    const finishSetup = () => {
      console.log("finish");
    };

    return {
      state,
      stepList,
      selectVCS,
      selectStep,
      finishSetup,
    };
  },
};
</script>
