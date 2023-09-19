<template>
  <RepositoryForm
    :create="true"
    :vcs-type="config.vcs.type"
    :vcs-name="config.vcs.title"
    :repository-info="config.repositoryInfo"
    :repository-config="config.repositoryConfig"
    :schema-change-type="config.schemaChangeType"
    :project="project"
    @change-schema-change-type="
      (type: SchemaChange) => $emit('change-schema-change-type', type)
    "
  />
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { Project, SchemaChange } from "@/types/proto/v1/project_service";
import { ProjectRepositoryConfig } from "../types";

defineProps({
  config: {
    required: true,
    type: Object as PropType<ProjectRepositoryConfig>,
  },
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

defineEmits<{
  (event: "change-schema-change-type", schemaChange: SchemaChange): void;
}>();
</script>
