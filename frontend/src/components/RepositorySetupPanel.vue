<template>
  <nav aria-label="Repository setup">
    <ol class="flex space-y-0 space-x-8">
      <li v-for="(step, index) in stepList" :key="index" class="flex-1">
        <!-- Completed Step -->
        <div
          class="group flex flex-col pt-4 border-t-4"
          :class="
            state.currentStep >= index
              ? 'border-accent hover:border-accent-hover cursor-pointer'
              : 'border-control-border'
          "
          @click.prevent="
            () => {
              if (state.currentStep >= index) {
                switchStep(index);
              }
            }
          "
        >
          <span
            class="text-xs font-semibold tracking-wide uppercase"
            :class="
              state.currentStep >= index
                ? 'text-accent group-hover:text-accent-hover'
                : 'text-control-light'
            "
            >Step {{ index + 1 }}</span
          >
          <span class="text-sm font-medium">{{ step }}</span>
        </div>
      </li>
    </ol>
  </nav>
  <template v-if="state.currentStep == SetupStep.CHOOSE_PROVIDER">
    choose provider
  </template>
  <template v-else-if="state.currentStep == SetupStep.SELECT_REPO">
    select repo
  </template>
  <template v-else-if="state.currentStep == SetupStep.CONFIGURE_DEPLOY">
    configure deploy
  </template>
  <div class="pt-4 flex justify-between">
    <button type="button" class="btn-normal" @click.prevent="cancel">
      Cancel
    </button>
    <div class="flex flex-row space-x-2">
      <button
        v-if="state.currentStep != 0"
        type="button"
        class="btn-normal"
        @click.prevent="switchStep(state.currentStep - 1)"
      >
        <svg
          class="-ml-1 mr-1 h-5 w-5 text-control-light"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M15 19l-7-7 7-7"
          ></path>
        </svg>
        <span>Back</span>
      </button>
      <button
        v-if="state.currentStep == stepList.length - 1"
        type="button"
        class="btn-primary"
        @click.prevent="finishSetup"
      >
        Finish
      </button>
      <button
        v-else
        type="button"
        class="btn-primary"
        @click.prevent="switchStep(state.currentStep + 1)"
      >
        Next
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
enum SetupStep {
  CHOOSE_PROVIDER = 0,
  SELECT_REPO = 1,
  CONFIGURE_DEPLOY = 2,
}

interface LocalState {
  currentStep: SetupStep;
}

const stepList = [
  "Choose Git provider",
  "Select repository",
  "Configure deploy",
];

export default {
  name: "RepositorySetupPanel",
  components: {},
  setup(props, ctx) {
    const state = reactive<LocalState>({
      currentStep: SetupStep.CHOOSE_PROVIDER,
    });

    const switchStep = (step: SetupStep) => {
      state.currentStep = step;
    };

    const finishSetup = () => {
      console.log("finish");
    };

    return {
      SetupStep,
      state,
      stepList,
      switchStep,
      finishSetup,
    };
  },
};
</script>
