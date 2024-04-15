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
import type { Project } from "@/types/proto/v1/project_service";
import type { ProjectRepositoryConfig } from "../types";

defineProps<{
  config: ProjectRepositoryConfig;
  project: Project;
}>();
</script>
