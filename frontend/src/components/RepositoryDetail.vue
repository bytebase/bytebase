<template>
  <RepositoryForm
    :vcsType="repository.vcs.type"
    :vcsName="repository.vcs.name"
    :repositoryInfo="repositoryInfo"
    :repositoryConfig="repositoryConfig"
  />
</template>

<script lang="ts">
import { PropType } from "vue";
import { useRouter } from "vue-router";
import RepositoryForm from "./RepositoryForm.vue";
import { Repository, ExternalRepositoryInfo, RepositoryConfig } from "../types";

export default {
  name: "RepositoryDetail",
  components: { RepositoryForm },
  props: {
    repository: {
      required: true,
      type: Object as PropType<Repository>,
    },
  },
  setup(props, ctx) {
    const router = useRouter();

    const repositoryInfo: ExternalRepositoryInfo = {
      externalId: props.repository.externalId,
      name: props.repository.name,
      fullPath: props.repository.fullPath,
      webURL: props.repository.webURL,
      defaultBranch: "",
    };

    const repositoryConfig: RepositoryConfig = {
      baseDirectory: props.repository.baseDirectory,
      branchFilter: props.repository.branchFilter,
    };

    return { repositoryInfo, repositoryConfig };
  },
};
</script>
