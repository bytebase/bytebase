<template>
  <h2 class="sr-only">Description</h2>
  <div class="prose max-w-none whitespace-pre-line" ref="taskContent">
    {{ task.attributes.content }}
  </div>
</template>

<script lang="ts">
import { nextTick, onMounted, PropType, ref } from "vue";
import { Task } from "../types";

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
    onMounted(() => {
      nextTick(() => {
        // Set focus
        // taskContent.value.focus();
      });
    });

    return {
      taskContent,
    };
  },
};
</script>
