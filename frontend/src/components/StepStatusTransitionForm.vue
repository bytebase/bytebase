<template>
  <div class="px-4 space-y-6 divide-y divide-gray-200">
    <div class="mt-2 grid grid-cols-1 gap-x-4 sm:grid-cols-4">
      <div class="sm:col-span-4 w-112 min-w-full">
        <label for="about" class="textlabel"> Note </label>
        <div class="mt-1">
          <textarea
            ref="commentTextArea"
            rows="3"
            class="textarea block w-full resize-none mt-1 text-sm text-control rounded-md whitespace-pre-wrap"
            placeholder="(Optional) Add a note..."
            v-model="state.comment"
            @input="
              (e) => {
                sizeToFit(e.target);
              }
            "
            @focus="
              (e) => {
                sizeToFit(e.target);
              }
            "
          ></textarea>
        </div>
      </div>
    </div>

    <!-- Update button group -->
    <div class="flex justify-end items-center pt-5">
      <button
        type="button"
        class="btn-normal mt-3 px-4 py-2 sm:mt-0 sm:w-auto"
        @click.prevent="$emit('cancel')"
      >
        No
      </button>
      <button
        type="button"
        class="ml-3 px-4 py-2"
        v-bind:class="submitButtonStyle"
        @click.prevent="$emit('submit', transition, state.comment)"
      >
        {{ okText }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive, ref, PropType } from "vue";
import { Issue } from "../types";
import { StepStatusTransition } from "../utils";

interface LocalState {
  comment: string;
}

export default {
  name: "StepStatusTransitionForm",
  emits: ["submit", "cancel"],
  props: {
    okText: {
      type: String,
      default: "OK",
    },
    issue: {
      required: true,
      type: Object as PropType<Issue>,
    },
    transition: {
      required: true,
      type: Object as PropType<StepStatusTransition>,
    },
  },
  setup(props, ctx) {
    const commentTextArea = ref("");

    const state = reactive<LocalState>({
      comment: "",
    });

    const submitButtonStyle = computed(() => {
      switch (props.transition.type) {
        case "CANCEL":
          return "btn-danger";
        default:
          return "btn-primary";
      }
    });

    return {
      state,
      commentTextArea,
      submitButtonStyle,
    };
  },
};
</script>
