<template>
  <div class="text-lg leading-6 font-medium text-control">Base directory</div>
  <BBTextField
    class="mt-2 w-full"
    :value="config.baseDirectory"
    @input="config.baseDirectory = $event.target.value"
  />
  <div class="mt-4 text-lg leading-6 font-medium text-control">Branch</div>
  <BBTextField
    class="mt-2 w-full"
    :placeholder="'Default branch (normally \'master\' or \'main\')'"
    :value="config.branch"
    @input="config.branch = $event.target.value"
  />
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import { PropType } from "@vue/runtime-core";
import { ProjectRepoConfig } from "../types";

interface LocalState {}

export default {
  name: "RepositoryConfigPanel",
  emits: ["next"],
  props: {
    config: {
      required: true,
      type: Object as PropType<ProjectRepoConfig>,
    },
  },
  components: {},
  setup(props, { emit }) {
    const state = reactive<LocalState>({});

    return {
      state,
    };
  },
};
</script>
