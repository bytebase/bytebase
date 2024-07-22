<template>
  <RepositoryForm
    :create="true"
    :vcs-type="config.vcs.type"
    :vcs-name="config.vcs.title"
    :repository-info="config.repositoryInfo"
    :repository-config="config.repositoryConfig"
    :project="project"
  />
</template>

<script lang="ts" setup>
import type { ComposedProject, ProjectRepositoryConfig } from "@/types";
import RepositoryForm from "./RepositoryForm.vue";

defineProps<{
  config: ProjectRepositoryConfig;
  project: ComposedProject;
}>();
</script>
