<template>
  <RepositoryForm
    :create="true"
    :vcsType="config.vcs.type"
    :vcsName="config.vcs.name"
    :repositoryInfo="config.repositoryInfo"
    :repositoryConfig="config.repositoryConfig"
  />
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import { PropType } from "@vue/runtime-core";
import RepositoryForm from "./RepositoryForm.vue";
import { ProjectRepositoryConfig } from "../types";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default {
  name: "RepositoryConfigPanel",
  components: { RepositoryForm },
  props: {
    config: {
      required: true,
      type: Object as PropType<ProjectRepositoryConfig>,
    },
  },
  emits: ["next"],
  setup() {
    const state = reactive<LocalState>({});

    return {
      state,
    };
  },
};
</script>
