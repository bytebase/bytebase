<template>
  <BBStepTab
    :stepItemList="stepList"
    @select-step="selectStep"
    @finish="finishSetup"
  >
    <template v-slot:0> choose provider </template>
    <template v-slot:1> select repo </template>
    <template v-slot:2> configure deploy </template>
  </BBStepTab>
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import { BBStepTabItem } from "../bbkit/types";

interface LocalState {}

const stepList: BBStepTabItem[] = [
  { title: "Choose Git provider" },
  { title: "Select repository" },
  { title: "Configure deploy" },
];

export default {
  name: "RepositorySetupPanel",
  components: {},
  setup(props, ctx) {
    const state = reactive<LocalState>({});

    const selectStep = (step: number) => {
      console.log("select step", step);
    };

    const finishSetup = () => {
      console.log("finish");
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
