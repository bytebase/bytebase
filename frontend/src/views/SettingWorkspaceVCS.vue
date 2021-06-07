<template>
  <h1>VCS</h1>
  <BBStepTab
    :stepItemList="stepList"
    @select-step="selectStep"
    @finish="finishSetup"
  >
    <template v-slot:0> choose git provider </template>
    <template v-slot:1> fill provider basic info </template>
    <template v-slot:2> configure settings </template>
  </BBStepTab>
</template>

<script lang="ts">
import { BBStepTabItem } from "../bbkit/types";

const stepList: BBStepTabItem[] = [
  { title: "Choose Git provider" },
  { title: "Fill provider basic info" },
  { title: "Configure settings" },
];

export default {
  name: "SettingWorkspaceVCS",
  props: {},
  setup(props, ctx) {
    const selectStep = (step: number) => {
      console.log("select step", step);
    };

    const finishSetup = () => {
      console.log("finish");
    };

    return {
      stepList,
      selectStep,
      finishSetup,
    };
  },
};
</script>
