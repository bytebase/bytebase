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

interface LocalState {}

export default {
  name: "RepositoryConfigPanel",
  emits: ["next"],
  props: {
    config: {
      required: true,
      type: Object as PropType<ProjectRepositoryConfig>,
    },
  },
  components: { RepositoryForm },
  setup(props, { emit }) {
    const state = reactive<LocalState>({});

    return {
      state,
    };
  },
};
</script>
