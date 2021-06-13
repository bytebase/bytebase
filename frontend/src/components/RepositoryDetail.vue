<template>
  <div class="text-lg leading-6 font-medium text-main">
    Version control is enabled
  </div>
  <div class="mt-1 textinfolabel">
    Database migration scripts are stored in
    <a class="normal-link" :href="repository.webURL" target="_blank">{{
      repository.fullPath
    }}</a
    >. To make schema changes, a developer would create a migration script and
    submit for review there. After the script is approved and merged into the
    branch matching pattern
    <span class="text-main">{{ repository.branchFilter }}</span
    >, Bytebase will automatically kicks off the task to apply the new schema
    change.
  </div>
  <RepositoryForm
    class="mt-4"
    :vcsType="repository.vcs.type"
    :vcsName="repository.vcs.name"
    :repositoryInfo="repositoryInfo"
    :repositoryConfig="repositoryConfig"
  />
  <div class="mt-4 pt-4 flex border-t justify-between">
    <BBButtonConfirm
      :style="'RESTORE'"
      :buttonText="'Restore to UI workflow'"
      :requireConfirm="true"
      :okText="'Restore'"
      :confirmTitle="'Restore to UI workflow?'"
      :confirmDescription="'When using the UI workflow, the developer submits a SQL review ticket directly from Bytebase and waits for the assigned DBA or peer developer to review. Bytebase applies the SQL change after review approved.'"
      @confirm="restoreToUIWorkflowType"
    />
    <div>
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        Cancel
      </button>
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowUpdate"
        @click.prevent="doUpdate"
      >
        Update
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { useRouter } from "vue-router";
import RepositoryForm from "./RepositoryForm.vue";
import {
  Repository,
  ExternalRepositoryInfo,
  RepositoryConfig,
  Project,
} from "../types";
import { useStore } from "vuex";

export default {
  name: "RepositoryDetail",
  components: { RepositoryForm },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
    repository: {
      required: true,
      type: Object as PropType<Repository>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const repositoryInfo = computed((): ExternalRepositoryInfo => {
      return {
        externalId: props.repository.externalId,
        name: props.repository.name,
        fullPath: props.repository.fullPath,
        webURL: props.repository.webURL,
        defaultBranch: "",
      };
    });

    const repositoryConfig = computed((): RepositoryConfig => {
      return {
        baseDirectory: props.repository.baseDirectory,
        branchFilter: props.repository.branchFilter,
      };
    });

    const allowUpdate = computed(() => {
      return true;
    });

    const restoreToUIWorkflowType = () => {
      store.dispatch(
        "repository/deleteRepositoryByProjectId",
        props.project.id
      );
    };

    const doUpdate = () => {};

    const cancel = () => {};

    return {
      repositoryInfo,
      repositoryConfig,
      allowUpdate,
      restoreToUIWorkflowType,
      doUpdate,
      cancel,
    };
  },
};
</script>
