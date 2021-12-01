<template>
  <div class="text-lg leading-6 font-medium text-main">
    Version control is <span class="text-success">enabled</span>
  </div>
  <div class="mt-2 textinfolabel">
    Database migration scripts are stored in
    <a class="normal-link" :href="repository.webUrl" target="_blank">{{
      repository.fullPath
    }}</a
    >. To make schema changes, a developer would create a migration script
    matching file path pattern
    <span class="font-medium text-main">{{
      state.repositoryConfig.baseDirectory
    }}</span>
    <span class="font-medium text-main"
      >/{{ state.repositoryConfig.filePathTemplate }}</span
    >. After the script is review approved and merged into the
    <template v-if="state.repositoryConfig.branchFilter"
      >branch matching pattern
      <span class="font-medium text-main">{{
        state.repositoryConfig.branchFilter
      }}</span></template
    >
    <span v-else class="font-medium text-main">default branch</span>, Bytebase
    will automatically kicks off the task to apply the new schema change.
    <template v-if="state.repositoryConfig.schemaPathTemplate"
      >After applying the schema change, if schema path template is specified,
      Bytebase will also write the latest schema to the specified schema path
      location
      <span class="font-medium text-main">{{
        state.repositoryConfig.schemaPathTemplate
      }}</span
      >.</template
    >
  </div>
  <RepositoryForm
    class="mt-4"
    :allow-edit="allowEdit"
    :vcs-type="repository.vcs.type"
    :vcs-name="repository.vcs.name"
    :repository-info="repositoryInfo"
    :repository-config="state.repositoryConfig"
    @change-repository="$emit('change-repository')"
  />
  <div v-if="allowEdit" class="mt-4 pt-4 flex border-t justify-between">
    <BBButtonConfirm
      :style="'RESTORE'"
      :button-text="'Restore to UI workflow'"
      :require-confirm="true"
      :ok-text="'Restore'"
      :confirm-title="'Restore to UI workflow?'"
      :confirm-description="'When using the UI workflow, the developer submits a SQL review ticket directly from Bytebase and waits for the assigned DBA or peer developer to review. Bytebase applies the SQL change after review approved.'"
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
import isEmpty from "lodash-es/isEmpty";
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
  name: "RepositoryPanel",
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
    allowEdit: {
      default: true,
      type: Boolean,
    },
  },
  emits: ["change-repository"],
  setup(props) {
    const store = useStore();
    const state = reactive<LocalState>({
      repositoryConfig: {
        baseDirectory: props.repository.baseDirectory,
        branchFilter: props.repository.branchFilter,
        filePathTemplate: props.repository.filePathTemplate,
        schemaPathTemplate: props.repository.schemaPathTemplate,
      },
    });

    watch(
      () => props.repository,
      (cur) => {
        state.repositoryConfig = {
          baseDirectory: cur.baseDirectory,
          branchFilter: cur.branchFilter,
          filePathTemplate: cur.filePathTemplate,
          schemaPathTemplate: cur.schemaPathTemplate,
        };
      }
    );

    const repositoryInfo = computed((): ExternalRepositoryInfo => {
      return {
        externalId: props.repository.externalId,
        name: props.repository.name,
        fullPath: props.repository.fullPath,
        webUrl: props.repository.webUrl,
      };
    });

    const allowUpdate = computed(() => {
      return (
        !isEmpty(state.repositoryConfig.branchFilter) &&
        !isEmpty(state.repositoryConfig.filePathTemplate) &&
        (props.repository.branchFilter != state.repositoryConfig.branchFilter ||
          props.repository.baseDirectory !=
            state.repositoryConfig.baseDirectory ||
          props.repository.filePathTemplate !=
            state.repositoryConfig.filePathTemplate ||
          props.repository.schemaPathTemplate !=
            state.repositoryConfig.schemaPathTemplate)
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
        props.repository.branchFilter != state.repositoryConfig.branchFilter
      ) {
        repositoryPatch.branchFilter = state.repositoryConfig.branchFilter;
      }
      if (
        props.repository.baseDirectory != state.repositoryConfig.baseDirectory
      ) {
        repositoryPatch.baseDirectory = state.repositoryConfig.baseDirectory;
      }
      if (
        props.repository.filePathTemplate !=
        state.repositoryConfig.filePathTemplate
      ) {
        repositoryPatch.filePathTemplate =
          state.repositoryConfig.filePathTemplate;
      }
      if (
        props.repository.schemaPathTemplate !=
        state.repositoryConfig.schemaPathTemplate
      ) {
        repositoryPatch.schemaPathTemplate =
          state.repositoryConfig.schemaPathTemplate;
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
