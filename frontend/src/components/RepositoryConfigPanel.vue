<template>
  <RepositoryForm
    :create="true"
    :vcs-type="config.vcs.type"
    :vcs-name="config.vcs.name"
    :repository-info="config.repositoryInfo"
    :repository-config="config.repositoryConfig"
    :schema-migration-type="config.schemaMigrationType"
    :project="project"
    @change-schema-migration-type="
      (type) => $emit('change-schema-migration-type', type)
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
  },
  emits: ["change-schema-migration-type"],
});
</script>
