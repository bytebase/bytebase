<template>
  <!-- Description Bar -->
  <div class="flex justify-between">
    <div class="textlabel">Description</div>
    <div v-if="!create" class="space-x-2">
      <button
        v-if="allowEdit && !state.editing"
        type="button"
        class="btn-icon"
        @click.prevent="beginEdit"
      >
        <!-- Heroicon name: solid/pencil -->
        <!-- Use h-5 to avoid flickering when show/hide icon -->
        <svg
          class="h-5 w-5"
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
      <!-- mt-0.5 is to prevent jiggling between switching edit/none-edit -->
      <button
        v-if="state.editing"
        type="button"
        class="
          mt-0.5
          px-3
          rounded-sm
          text-control
          hover:bg-control-bg-hover
          disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed
          text-sm
          leading-5
          font-normal
          focus:ring-control focus:outline-none
          focus-visible:ring-2
          focus:ring-offset-2
        "
        @click.prevent="cancelEdit"
      >
        Cancel
      </button>
      <button
        v-if="state.editing"
        type="button"
        class="
          mt-0.5
          px-3
          border border-control-border
          rounded-sm
          text-control
          bg-control-bg
          hover:bg-control-bg-hover
          disabled:bg-control-bg disabled:opacity-50 disabled:cursor-not-allowed
          text-sm
          leading-5
          font-normal
          focus:ring-control focus:outline-none
          focus-visible:ring-2
          focus:ring-offset-2
        "
        :disabled="state.editDescription == issue.description"
        @click.prevent="saveEdit"
      >
        Save
      </button>
    </div>
  </div>
  <!-- Description -->
  <label for="description" class="sr-only">Edit Description</label>
  <!-- Use border-white focus:border-white to have the invisible border width
      otherwise it will have 1px jiggling switching between focus/unfocus state -->
  <textarea
    ref="editDescriptionTextArea"
    v-model="state.editDescription"
    :rows="create ? 10 : 5"
    class="
      mt-2
      w-full
      resize-none
      whitespace-pre-wrap
      border-white
      focus:border-white
      outline-none
    "
    :class="state.editing ? 'focus:ring-control focus-visible:ring-2' : ''"
    :style="
      state.editing
        ? ''
        : '-webkit-box-shadow: none; -moz-box-shadow: none; box-shadow: none'
    "
    placeholder="Add some description..."
    :readonly="!state.editing"
    @input="
      (e) => {
        sizeToFit(e.target);
        // When creating the issue, we will emit the event on keystroke to update the in-memory state.
        if (create) {
          $emit('update-description', state.editDescription);
        }
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
import {
  nextTick,
  onMounted,
  onUnmounted,
  PropType,
  ref,
  reactive,
  watch,
} from "vue";
import { Issue } from "../types";
import { sizeToFit } from "../utils";

interface LocalState {
  editing: boolean;
  editDescription: string;
}

export default {
  name: "IssueDescriptionPanel",
  props: {
    issue: {
      required: true,
      type: Object as PropType<Issue>,
    },
    create: {
      required: true,
      type: Boolean,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  emits: ["update-description"],
  setup(props, { emit }) {
    const editDescriptionTextArea = ref();

    const state = reactive<LocalState>({
      editing: false,
      editDescription: props.issue.description,
    });

    const keyboardHandler = (e: KeyboardEvent) => {
      if (
        state.editing &&
        editDescriptionTextArea.value === document.activeElement
      ) {
        if (e.code == "Escape") {
          cancelEdit();
        } else if (e.code == "Enter" && e.metaKey) {
          if (state.editDescription != props.issue.description) {
            saveEdit();
          }
        }
      }
    };

    const resizeTextAreaHandler = () => {
      sizeToFit(editDescriptionTextArea.value);
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
      window.addEventListener("resize", resizeTextAreaHandler);
      if (props.create) {
        state.editing = true;
        sizeToFit(editDescriptionTextArea.value);
      }
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
      window.removeEventListener("resize", resizeTextAreaHandler);
    });

    // Reset the edit state after creating the issue.
    watch(
      () => props.create,
      (curNew, prevNew) => {
        if (!curNew && prevNew) {
          state.editing = false;
        }
      }
    );

    watch(
      () => props.issue,
      (curIssue) => {
        state.editDescription = curIssue.description;
        nextTick(() => {
          sizeToFit(editDescriptionTextArea.value);
        });
      }
    );

    const beginEdit = () => {
      state.editDescription = props.issue.description;
      state.editing = true;
      nextTick(() => {
        editDescriptionTextArea.value.focus();
      });
    };

    const saveEdit = () => {
      emit(
        "update-description",
        state.editDescription,
        (updatedIssue: Issue) => {
          state.editDescription = updatedIssue.description;
          state.editing = false;
          nextTick(() => {
            sizeToFit(editDescriptionTextArea.value);
          });
        }
      );
    };

    const cancelEdit = () => {
      state.editDescription = props.issue.description;
      state.editing = false;
      nextTick(() => {
        sizeToFit(editDescriptionTextArea.value);
      });
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
