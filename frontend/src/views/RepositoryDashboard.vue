<template>
  Repository
  {{ repository?.attributes.type }}
  <br />
  Access Token
  {{ repository?.attributes.comment.accessToken }}
</template>

<script lang="ts">
import { watchEffect, computed } from "vue";
import { useStore } from "vuex";
import { Project } from "../types";

export default {
  name: "RepositoryDashboard",
  props: {
    groupSlug: {
      required: true,
      type: String,
    },
    projectSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const project: Project = store.getters["project/projectByNamespaceAndSlug"](
      props.groupSlug,
      props.projectSlug
    );

    const prepareRepository = () => {
      store
        .dispatch("repository/fetchRepositoryForProject", project.id)
        .catch((error) => {
          console.log(error);
        });
    };

    const repository = computed(() =>
      store.getters["repository/repositoryByProject"](project.id)
    );

    watchEffect(prepareRepository);

    return {
      repository,
    };
  },
};
</script>
