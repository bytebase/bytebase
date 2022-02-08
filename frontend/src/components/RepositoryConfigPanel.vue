<template>
  <RepositoryForm
    :create="true"
    :vcs-type="config.vcs.type"
    :vcs-name="config.vcs.name"
    :repository-info="config.repositoryInfo"
    :repository-config="config.repositoryConfig"
    :project="project"
  />
</template>

<script lang="ts">
import { reactive, PropType, defineComponent } from "vue";
import RepositoryForm from "./RepositoryForm.vue";
import { Project, ProjectRepositoryConfig } from "../types";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
  name: "RepositoryConfigPanel",
  components: { RepositoryForm },
  props: {
    config: {
      required: true,
      type: Object as PropType<ProjectRepositoryConfig>,
    },
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  emits: ["next"],
  setup() {
    const state = reactive<LocalState>({});

    return {
      state,
    };
  },
});
</script>
