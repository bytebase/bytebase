<template>
  <div class="px-4 space-y-6 divide-y divide-gray-200">
    <div class="mt-2 grid grid-cols-1 gap-x-4 sm:grid-cols-4">
      <template v-if="transition.type == 'RESOLVE'">
        <template v-for="(field, index) in outputFieldList" :key="index">
          <div class="sm:col-span-1">
            <label class="textlabel">
              {{ field.name
              }}<span v-if="field.required" class="text-red-600">*</span>
            </label>
          </div>
          <div class="sm:col-span-4 sm:col-start-1">
            <div class="mt-1 flex rounded-md shadow-sm">
              <input
                type="text"
                :name="field.id"
                :id="field.id"
                v-model="state.outputValueList[index]"
                autocomplete="off"
                class="w-full textfield"
              />
            </div>
          </div>
          <div v-if="index == outputFieldList.length - 1" class="mt-4" />
        </template>
      </template>

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
        :disabled="!allowSubmit"
        @click.prevent="$emit('submit', state.outputValueList, state.comment)"
      >
        {{ okText }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, ref, PropType } from "vue";
import cloneDeep from "lodash-es/cloneDeep";
import { Task, TaskStatusTransition } from "../types";
import { TaskField } from "../plugins";

interface LocalState {
  comment: string;
  outputValueList: string[];
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
      required: true,
      type: Object as PropType<Task>,
    },
    transition: {
      required: true,
      type: Object as PropType<TaskStatusTransition>,
    },
    outputFieldList: {
      required: true,
      type: Object as PropType<TaskField[]>,
    },
  },
  setup(props, ctx) {
    const commentTextArea = ref("");

    const state = reactive<LocalState>({
      comment: "",
      outputValueList: [],
    });

    onMounted(() => {
      for (const field of props.outputFieldList) {
        state.outputValueList.push(cloneDeep(props.task.payload[field.id]));
      }
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

    const allowSubmit = computed(() => {
      if (props.transition.type != "RESOLVE") {
        return true;
      }
      for (let i = 0; i < props.outputFieldList.length; i++) {
        const field = props.outputFieldList[i];
        if (field.required && field.isEmpty(state.outputValueList[i])) {
          return false;
        }
      }
      return true;
    });

    return {
      state,
      commentTextArea,
      submitButtonStyle,
      allowSubmit,
    };
  },
};
</script>
