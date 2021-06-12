<template>
  <template v-if="state.showWizard">
    <RepositorySetupWizard
      :project="project"
      @cancel="cancelWizard"
      @finish="finishWizard"
    />
  </template>
  <template v-else>
    <!-- Use the persistent workflowType here -->
    <template v-if="project.workflowType == 'UI'">
      <div class="text-lg leading-6 font-medium text-main">
        Current workflow
      </div>
      <div class="mt-4 flex flex-col space-y-4">
        <div class="flex space-x-4">
          <input
            name="UI workflow"
            tabindex="-1"
            type="radio"
            class="btn"
            value="UI"
            v-model="state.workflowType"
          />
          <div class="-mt-0.5">
            <div class="textlabel">UI workflow (no version control)</div>
            <div class="mt-1 textinfolabel">
              Classic SQL Review workflow where the developer submits a SQL
              review ticket directly from Bytebase and waits for the assigned
              DBA or peer developer to review. Bytebase applies the SQL change
              after review approved.
            </div>
          </div>
        </div>
        <div class="flex space-x-4">
          <input
            name="Version control workflow"
            tabindex="-1"
            type="radio"
            class="btn"
            value="VCS"
            v-model="state.workflowType"
          />
          <div class="-mt-0.5">
            <div class="textlabel">Version control workflow</div>
            <div class="mt-1 textinfolabel">
              Database migration scripts are stored in a git repository. To make
              schema changes, a developer would create a migration script and
              submit for review in the corresponding VCS such as GitLab. After
              the script is approved and merged into the configured branch,
              Bytebase will automatically kicks off the task to apply the new
              schema change.
            </div>
          </div>
        </div>
      </div>
      <template v-if="state.workflowType == 'VCS'">
        <div class="mt-4 flex items-center justify-end">
          <button
            type="button"
            class="btn-primary inline-flex justify-center py-2 px-2"
            @click.prevent="enterWizard"
          >
            Config workflow
            <svg
              class="ml-1 w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9 5l7 7-7 7"
              ></path>
            </svg>
          </button>
        </div>
      </template>
    </template>
    <template v-else-if="project.workflowType == 'VCS'">
      <RepositoryDetail :repository="repository" />
    </template>
  </template>
</template>

<script lang="ts">
import { reactive, watchEffect, watch } from "vue";
import { computed, PropType } from "@vue/runtime-core";
import RepositorySetupWizard from "./RepositorySetupWizard.vue";
import RepositoryDetail from "./RepositoryDetail.vue";
import { Project, ProjectWorkflowType, Repository, UNKNOWN_ID } from "../types";
import { useStore } from "vuex";

interface LocalState {
  workflowType: ProjectWorkflowType;
  showWizard: boolean;
}

export default {
  name: "ProjectVCSPanel",
  components: {
    RepositorySetupWizard,
    RepositoryDetail,
  },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  async setup(props) {
    const store = useStore();
    const state = reactive<LocalState>({
      workflowType: props.project.workflowType,
      showWizard: false,
    });

    const prepareRepository = () => {
      store.dispatch("repository/fetchRepositoryByProjectId", props.project.id);
    };

    watchEffect(prepareRepository);

    watch(
      () => props.project,
      (cur, _) => {
        state.workflowType = cur.workflowType;
      }
    );

    const repository = computed((): Repository => {
      return store.getters["repository/repositoryByProjectId"](
        props.project.id
      );
    });

    const enterWizard = () => {
      state.showWizard = true;
    };

    const cancelWizard = () => {
      state.workflowType = "UI";
      state.showWizard = false;
    };

    const finishWizard = () => {
      state.showWizard = false;
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "SUCCESS",
        title: `Successfully enabled version control workflow for '${props.project.name}'`,
      });
    };

    return {
      state,
      UNKNOWN_ID,
      repository,
      enterWizard,
      cancelWizard,
      finishWizard,
    };
  },
};
</script>
