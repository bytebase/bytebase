<template>
  <!-- Content Bar -->
  <div class="flex justify-end space-x-3">
    <button
      v-if="!state.edit"
      type="button"
      class="btn-normal"
      @click.prevent="beginEdit"
    >
      <!-- Heroicon name: solid/pencil -->
      <svg
        class="-ml-1 mr-2 h-5 w-5 text-gray-400"
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
      >
        <path
          d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
        />
      </svg>
      <span>Edit</span>
    </button>
    <button
      v-if="state.edit"
      type="button"
      class="btn-cancel"
      @click.prevent="cancelEdit"
    >
      Cancel
    </button>
    <button
      v-if="state.edit"
      type="button"
      class="btn-normal"
      @click.prevent="saveEdit"
    >
      Save
    </button>
  </div>
  <!-- Content -->
  <BBAutoResize>
    <template v-slot:main="{ resize }">
      <label for="content" class="sr-only">Edit Description</label>
      <!-- Use border-white focus:border-white to have the invisible border width
      otherwise it will have 1px jiggling switching between focus/unfocus state -->
      <textarea
        ref="editContentTextArea"
        rows="10"
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
            resize(e.target);
          }
        "
        @focus="
          (e) => {
            resize(e.target);
          }
        "
      ></textarea>
    </template>
  </BBAutoResize>
</template>

<script lang="ts">
import { nextTick, onMounted, PropType, ref, reactive } from "vue";
import { useStore } from "vuex";
import { Task } from "../types";

interface LocalState {
  edit: boolean;
}

export default {
  name: "TaskContent",
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
  },
  components: {},
  setup(props, ctx) {
    const store = useStore();
    const editContent = ref(props.task.attributes.content);
    const editContentTextArea = ref();

    const state = reactive<LocalState>({
      edit: false,
    });

    const beginEdit = () => {
      editContent.value = props.task.attributes.content;
      state.edit = true;
      nextTick(() => {
        editContentTextArea.value.focus();
      });
    };

    const saveEdit = () => {
      store
        .dispatch("task/patchTask", {
          taskId: props.task.id,
          taskPatch: {
            content: editContent.value,
          },
        })
        .then((updatedTask: Task) => {
          state.edit = false;
          editContent.value = updatedTask.attributes.content;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const cancelEdit = () => {
      state.edit = false;
      editContent.value = props.task.attributes.content;
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
