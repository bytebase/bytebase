<template>
  <RepositoryForm
    :create="true"
    :vcs-type="config.vcs.type"
    :vcs-name="config.vcs.name"
    :repository-info="config.repositoryInfo"
    :repository-config="config.repositoryConfig"
    :schema-change-type="config.schemaChangeType"
    :project="project"
    :form-error="formError"
    @change-schema-change-type="
      (type) => $emit('change-schema-change-type', type)
    "
  />
</template>

<script lang="ts">
import { PropType, defineComponent } from "vue";
import RepositoryForm from "./RepositoryForm.vue";
import { Project, ProjectRepositoryConfig } from "../types";

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
    formError: {
      required: true,
      type: Object as PropType<Record<string, boolean>>,
    },
  },
  emits: ["change-schema-change-type"],
});
</script>
