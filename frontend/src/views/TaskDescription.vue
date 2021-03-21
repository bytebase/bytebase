<template>
  <!-- Description Bar -->
  <div v-if="!$props.new" class="flex justify-end space-x-2">
    <button
      v-if="!state.edit"
      type="button"
      class="btn-icon"
      @click.prevent="beginEdit"
    >
      <!-- Heroicon name: solid/pencil -->
      <svg
        class="h-6 w-6"
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
      >
        <path
          d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
        />
      </svg>
    </button>
    <!-- mt-0.5 is to prevent jiggling betweening switching edit/none-edit -->
    <button
      v-if="state.edit"
      type="button"
      class="mt-0.5 px-3 rounded-sm text-control hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed text-sm leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
      @click.prevent="cancelEdit"
    >
      Cancel
    </button>
    <button
      v-if="state.edit"
      type="button"
      class="mt-0.5 px-3 border border-control-border rounded-sm text-control bg-control-bg hover:bg-control-bg-hover disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed text-sm leading-5 font-normal focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
      :disabled="state.editDescription == task.description"
      @click.prevent="saveEdit"
    >
      Save
    </button>
  </div>
  <!-- Description -->
  <label for="description" class="sr-only">Edit Description</label>
  <!-- TODO: Has control flickering switching between edit/non-edit mode -->
  <template v-if="state.edit">
    <textarea
      ref="editDescriptionTextArea"
      :rows="$props.new ? 10 : 5"
      class="whitespace-pre-wrap mt-2 w-full focus:ring-control focus-visible:ring-2 resize-none border-white focus:border-white outline-none"
      placeholder="Add some description..."
      v-model="state.editDescription"
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
  </template>
  <div
    v-else
    class="mt-4"
    style="margin-left: 5px; margin-top: 9px; margin-bottom: 23px"
  >
    <div v-highlight class="whitespace-pre-wrap">
      {{ state.editDescription }}
    </div>
  </div>
</template>

<script lang="ts">
import {
  nextTick,
  onMounted,
  onUnmounted,
  watch,
  PropType,
  ref,
  reactive,
} from "vue";
import { Task } from "../types";
import { sizeToFit } from "../utils";

interface LocalState {
  edit: boolean;
  editDescription: string;
}

export default {
  name: "TaskDescription",
  emits: ["update-description"],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
    new: {
      required: true,
      type: Boolean,
    },
  },
  components: {},
  setup(props, { emit }) {
    const editDescriptionTextArea = ref();

    const state = reactive<LocalState>({
      edit: false,
      editDescription: props.task.description,
    });

    const keyboardHandler = (e: KeyboardEvent) => {
      if (
        state.edit &&
        editDescriptionTextArea.value === document.activeElement
      ) {
        if (e.code == "Escape") {
          cancelEdit();
        } else if (e.code == "Enter" && e.metaKey) {
          if (state.editDescription != props.task.description) {
            saveEdit();
          }
        }
      }
    };

    const resizeTextAreaHandler = () => {
      if (state.edit) {
        sizeToFit(editDescriptionTextArea.value);
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
      window.addEventListener("resize", resizeTextAreaHandler);
      if (props.new) {
        state.edit = true;
      }
      nextTick(() => {
        if (props.new) {
          editDescriptionTextArea.value.focus();
          sizeToFit(editDescriptionTextArea.value);
        }
      });
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
      window.removeEventListener("resize", resizeTextAreaHandler);
    });

    // Reset the edit state after creating the task.
    watch(
      () => props.new,
      (curNew, prevNew) => {
        if (!curNew && prevNew) {
          state.edit = false;
        }
      }
    );

    watch(
      () => props.task,
      (curTask, prevTask) => {
        state.editDescription = curTask.description;
        nextTick(() => {
          if (state.edit) {
            sizeToFit(editDescriptionTextArea.value);
          }
        });
      }
    );

    const beginEdit = () => {
      state.editDescription = props.task.description;
      state.edit = true;
      nextTick(() => {
        editDescriptionTextArea.value.focus();
      });
    };

    const saveEdit = () => {
      emit("update-description", state.editDescription, (updatedTask: Task) => {
        state.editDescription = updatedTask.description;
        state.edit = false;
      });
    };

    const cancelEdit = () => {
      state.editDescription = props.task.description;
      state.edit = false;
    };

    return {
      state,
      editDescriptionTextArea,
      beginEdit,
      saveEdit,
      cancelEdit,
    };
  },
};
</script>
