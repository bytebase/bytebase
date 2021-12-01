<template>
  <div>
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
                changeStep(index);
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
      </li>
    </ol>
    <div class="mt-4 mb-4">
      <template v-for="(step, index) in stepItemList" :key="index">
        <slot
          v-if="state.currentStep == index"
          :name="index"
          :next="
            () => {
              changeStep(state.currentStep + 1);
            }
          "
        />
      </template>
    </div>
    <div class="pt-4 border-t border-block-border flex justify-between">
      <button type="button" class="btn-normal" @click.prevent="cancel">
        Cancel
      </button>
      <div class="flex flex-row space-x-2">
        <button
          v-if="state.currentStep != 0"
          type="button"
          class="btn-normal"
          @click.prevent="changeStep(state.currentStep - 1)"
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
          :disabled="!allowNext"
          type="button"
          class="btn-primary"
          @click.prevent="finish"
        >
          {{ finishTitle }}
        </button>
        <button
          v-else-if="!stepItemList[state.currentStep].hideNext"
          :disabled="!allowNext"
          type="button"
          class="btn-primary"
          @click.prevent="changeStep(state.currentStep + 1)"
        >
          Next
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { PropType, reactive } from "vue";
import { BBStepTabItem } from "./types";

interface LocalState {
  done: boolean;
  currentStep: number;
}

export default {
  name: "BBStepTab",
  props: {
    stepItemList: {
      required: true,
      type: Object as PropType<BBStepTabItem[]>,
    },
    allowNext: {
      default: true,
      type: Boolean,
    },
    finishTitle: {
      default: "Finish",
      type: String,
    },
  },
  // For try-change-step and try-finish listener, it needs to call the callback if it determines we can change the step.
  emits: ["try-change-step", "try-finish", "cancel"],
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      done: false,
      currentStep: 0,
    });

    const changeStep = (step: number) => {
      const changeStepCallback = () => {
        state.done = false;
        state.currentStep = step;
      };
      emit("try-change-step", state.currentStep, step, changeStepCallback);
    };

    const finish = () => {
      const finishCallback = () => {
        state.done = true;
      };
      emit("try-finish", finishCallback);
    };

    const cancel = () => {
      emit("cancel");
    };

    return {
      state,
      changeStep,
      finish,
      cancel,
    };
  },
};
</script>
