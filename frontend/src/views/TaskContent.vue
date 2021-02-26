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
  <h2 class="sr-only">Description</h2>
  <div class="mt-2 prose max-w-none whitespace-pre-line" ref="taskContent">
    {{ task.attributes.content }}
  </div>
</template>

<script lang="ts">
import { nextTick, onMounted, PropType, ref, reactive } from "vue";
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
    const taskContent = ref(null);

    const state = reactive<LocalState>({
      edit: false,
    });

    onMounted(() => {
      nextTick(() => {
        // Set focus
        // taskContent.value.focus();
      });
    });

    const beginEdit = () => {
      state.edit = true;
    };

    const saveEdit = () => {
      state.edit = false;
    };

    const cancelEdit = () => {
      state.edit = false;
    };

    return {
      state,
      taskContent,
      beginEdit,
      saveEdit,
      cancelEdit,
    };
  },
};
</script>
