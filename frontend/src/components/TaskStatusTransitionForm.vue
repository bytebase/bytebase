<template>
  <div class="px-4 space-y-6 divide-y divide-gray-200">
    <div class="mt-4 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
      <div class="sm:col-span-6">
        <label for="about" class="block text-sm font-medium text-gray-700">
          Note
        </label>
        <div class="mt-1">
          <textarea
            ref="commentTextArea"
            rows="3"
            class="textarea block w-full resize-none mt-1 text-sm text-control whitespace-pre-wrap"
            placeholder="Add an optional note..."
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
        @click.prevent="$emit('submit', state.comment)"
      >
        {{ okText }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive, ref, PropType } from "vue";
import { Task, TaskStatusTransition } from "../types";

interface LocalState {
  comment: string;
}

export default {
  name: "TaskStatusTransitionForm",
  emits: ["submit", "cancel"],
  props: {
    okText: {
      type: String,
      default: "OK",
    },
    task: {
      // Can be false when create is true
      required: false,
      type: Object as PropType<Task>,
    },
    transition: {
      required: true,
      type: Object as PropType<TaskStatusTransition>,
    },
  },
  setup(props, ctx) {
    const commentTextArea = ref("");

    const state = reactive<LocalState>({
      comment: "",
    });

    const submitButtonStyle = computed(() => {
      switch (props.transition.to) {
        case "OPEN":
          return "btn-primary";
        case "DONE":
          return "btn-success";
        case "CANCELED":
          return "btn-danger";
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
