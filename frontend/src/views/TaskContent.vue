<template>
  <!-- Content Bar -->
  <div class="flex justify-end space-x-2">
    <button
      v-if="!state.edit"
      type="button"
      class="btn-icon"
      @click.prevent="beginEdit"
    >
      <!-- Heroicon name: solid/pencil -->
      <svg
        class="h-6 w-6 text-control"
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
      @click.prevent="saveEdit"
    >
      Save
    </button>
  </div>
  <!-- Content -->
  <label for="content" class="sr-only">Edit Description</label>
  <!-- Use border-white focus:border-white to have the invisible border width
      otherwise it will have 1px jiggling switching between focus/unfocus state -->
  <textarea
    ref="editContentTextArea"
    rows="5"
    class="mt-4 rounded-md w-full resize-none whitespace-pre-line border-white focus:border-white outline-none"
    :class="
      state.edit
        ? 'focus:ring-control focus-visible:ring-2 focus:ring-offset-4'
        : ''
    "
    :style="
      state.edit
        ? ''
        : '-webkit-box-shadow: none; -moz-box-shadow: none; box-shadow: none'
    "
    placeholder="Add some description..."
    :readonly="!state.edit"
    v-model="editContent"
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

<script lang="ts">
import { nextTick, onMounted, onUnmounted, PropType, ref, reactive } from "vue";
import { Task } from "../types";
import { sizeToFit } from "../utils";

interface LocalState {
  edit: boolean;
}

export default {
  name: "TaskContent",
  emits: ["update-content"],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
  },
  components: {},
  setup(props, { emit }) {
    const editContent = ref(props.task.attributes.content);
    const editContentTextArea = ref();

    const state = reactive<LocalState>({
      edit: false,
    });

    const keyboardHandler = (e: KeyboardEvent) => {
      if (state.edit && editContentTextArea.value === document.activeElement) {
        if (e.code == "Escape") {
          cancelEdit();
        } else if (e.code == "Enter" && e.metaKey) {
          saveEdit();
        }
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
      nextTick(() => {
        sizeToFit(editContentTextArea.value);
      });
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
    });

    const beginEdit = () => {
      editContent.value = props.task.attributes.content;
      state.edit = true;
      nextTick(() => {
        editContentTextArea.value.focus();
      });
    };

    const saveEdit = () => {
      emit("update-content", editContent.value, (updatedTask: Task) => {
        state.edit = false;
        editContent.value = updatedTask.attributes.content;
        nextTick(() => {
          sizeToFit(editContentTextArea.value);
        });
      });
    };

    const cancelEdit = () => {
      state.edit = false;
      editContent.value = props.task.attributes.content;
      nextTick(() => {
        sizeToFit(editContentTextArea.value);
      });
    };

    return {
      state,
      editContent,
      editContentTextArea,
      beginEdit,
      saveEdit,
      cancelEdit,
    };
  },
};
</script>
