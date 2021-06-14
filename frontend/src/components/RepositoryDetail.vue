<template>
  <div class="text-lg leading-6 font-medium text-main">
    Version control is <span class="text-success">enabled</span>
  </div>
  <div class="mt-2 textinfolabel">
    Database migration scripts are stored in
    <a class="normal-link" :href="repository.webURL" target="_blank">{{
      repository.fullPath
    }}</a
    >. To make schema changes, a developer would create a migration script under
    base directory
    <span class="font-medium text-main"
      >/{{ state.repositoryConfig.baseDirectory }}</span
    >. After the script is review approved and merged into the
    <template v-if="state.repositoryConfig.branchFilter"
      >branch matching pattern
      <span class="font-medium text-main">{{
        state.repositoryConfig.branchFilter
      }}</span></template
    >
    <span v-else class="font-medium text-main">default branch</span>, Bytebase
    will automatically kicks off the task to apply the new schema change.
  </div>
  <RepositoryForm
    class="mt-4"
    :vcsType="repository.vcs.type"
    :vcsName="repository.vcs.name"
    :repositoryInfo="repositoryInfo"
    :repositoryConfig="state.repositoryConfig"
    @change-repository="$emit('change-repository')"
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
import { computed, PropType, reactive, watch } from "vue";
import RepositoryForm from "./RepositoryForm.vue";
import {
  Repository,
  RepositoryPatch,
  ExternalRepositoryInfo,
  RepositoryConfig,
  Project,
} from "../types";
import { useStore } from "vuex";

interface LocalState {
  repositoryConfig: RepositoryConfig;
}

export default {
  name: "RepositoryDetail",
  emits: ["change-repository"],
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
    const state = reactive<LocalState>({
      repositoryConfig: {
        baseDirectory: props.repository.baseDirectory,
        branchFilter: props.repository.branchFilter,
      },
    });

    watch(
      () => props.repository,
      (cur, _) => {
        state.repositoryConfig = {
          baseDirectory: cur.baseDirectory,
          branchFilter: cur.branchFilter,
        };
      }
    );

    const repositoryInfo = computed((): ExternalRepositoryInfo => {
      return {
        externalId: props.repository.externalId,
        name: props.repository.name,
        fullPath: props.repository.fullPath,
        webURL: props.repository.webURL,
      };
    });

    const allowUpdate = computed(() => {
      return (
        props.repository.baseDirectory !=
          state.repositoryConfig.baseDirectory ||
        props.repository.branchFilter != state.repositoryConfig.branchFilter
      );
    });

    const restoreToUIWorkflowType = () => {
      store
        .dispatch("repository/deleteRepositoryByProjectId", props.project.id)
        .then(() => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully restored to UI workflow`,
          });
        });
    };

    const doUpdate = () => {
      const repositoryPatch: RepositoryPatch = {};
      if (
        props.repository.baseDirectory != state.repositoryConfig.baseDirectory
      ) {
        repositoryPatch.baseDirectory = state.repositoryConfig.baseDirectory;
      }
      if (
        props.repository.branchFilter != state.repositoryConfig.branchFilter
      ) {
        repositoryPatch.branchFilter = state.repositoryConfig.branchFilter;
      }
      store
        .dispatch("repository/updateRepositoryByProjectId", {
          projectId: props.project.id,
          repositoryPatch,
        })
        .then(() => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully updated version control config`,
          });
        });
    };

    return {
      state,
      repositoryInfo,
      allowUpdate,
      restoreToUIWorkflowType,
      doUpdate,
    };
  },
};
</script>
