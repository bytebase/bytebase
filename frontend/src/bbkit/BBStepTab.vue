<template>
  <ol class="flex space-y-0 space-x-8">
    <li v-for="(step, index) in stepItemList" :key="index" class="flex-1">
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
        <div class="flex items-center justify-between">
          <div class="flex flex-col">
            <span
              class="text-xs font-semibold tracking-wide uppercase"
              :class="
                state.currentStep >= index
                  ? 'text-accent group-hover:text-accent-hover'
                  : 'text-control-light'
              "
              >Step {{ index + 1 }}</span
            >
            <span class="text-sm font-medium">{{ step.title }}</span>
          </div>
          <div
            v-if="state.currentStep > index || state.done"
            class="
              flex
              items-center
              justify-center
              w-6
              h-6
              bg-accent
              text-white
              rounded-full
              select-none
            "
          >
            <svg
              class="w-4 h-4"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fill-rule="evenodd"
                d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                clip-rule="evenodd"
              />
            </svg>
          </div>
        </div>
      </div>
      <slot v-if="state.currentStep == index" :name="index" />
    </li>
  </ol>
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
        v-if="state.currentStep == stepItemList.length - 1"
        type="button"
        class="btn-primary"
        @click.prevent="finish"
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
import { PropType, reactive } from "@vue/runtime-core";
import { BBStepTabItem } from "./types";

interface LocalState {
  done: boolean;
  currentStep: number;
}

export default {
  name: "BBStepTab",
  emits: ["select-step", "finish"],
  props: {
    stepItemList: {
      required: true,
      type: Object as PropType<BBStepTabItem[]>,
    },
  },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      done: false,
      currentStep: 0,
    });

    const switchStep = (step: number) => {
      state.done = false;
      state.currentStep = step;
      emit("select-step", step);
    };

    const finish = () => {
      state.done = true;
      emit("finish");
    };

    return {
      state,
      switchStep,
      finish,
    };
  },
};
</script>
