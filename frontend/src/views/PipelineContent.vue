<template>
  <h2 class="sr-only">Description</h2>
  <div class="prose max-w-none whitespace-pre-line">
    {{ pipeline.attributes.content }}
  </div>
</template>

<script lang="ts">
import { PropType } from "vue";
import { Pipeline } from "../types";

export default {
  name: "PipelineContent",
  props: {
    pipeline: {
      required: true,
      type: Object as PropType<Pipeline>,
    },
  },
  components: {},
  setup(props, ctx) {},
};
</script>
