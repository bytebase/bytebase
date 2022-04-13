<template>
  <div>
    <template
      v-if="
        allowEdit && (state.showWizardForCreate || state.showWizardForChange)
      "
    >
      <RepositorySetupWizard
        :create="state.showWizardForCreate"
        :project="project"
        @cancel="cancelWizard"
        @finish="finishWizard"
      />
    </template>
    <template v-else>
      <!-- Use the persistent workflowType here -->
      <template v-if="project.workflowType == 'UI'">
        <div class="text-lg leading-6 font-medium text-main">
          {{ $t("workflow.current-workflow") }}
        </div>
        <div class="mt-6 flex flex-col space-y-4">
          <div class="flex space-x-4">
            <input
              id="workflow-ui"
              v-model="state.workflowType"
              name="UI workflow"
              tabindex="-1"
              type="radio"
              class="text-accent disabled:text-accent-disabled focus:ring-accent"
              value="UI"
              :disabled="!allowEdit"
            />
            <div class="-mt-1">
              <label for="workflow-ui" class="textlabel">{{
                $t("workflow.ui-workflow")
              }}</label>
              <div class="mt-1 textinfolabel">
                {{ $t("workflow.ui-workflow-description") }}
              </div>
            </div>
          </div>
          <div class="flex space-x-4">
            <input
              id="workflow-version-control"
              v-model="state.workflowType"
              name="Version control workflow"
              tabindex="-1"
              type="radio"
              class="text-accent disabled:text-accent-disabled focus:ring-accent"
              value="VCS"
              :disabled="!allowEdit"
            />
            <div class="-mt-1">
              <label for="workflow-version-control" class="textlabel">{{
                $t("workflow.gitops-workflow")
              }}</label>
              <div class="mt-1 textinfolabel">
                {{ $t("workflow.gitops-workflow-description") }}
              </div>
            </div>
          </div>
        </div>
        <template v-if="allowEdit && state.workflowType == 'VCS'">
          <div class="mt-4 flex items-center justify-end">
            <button
              type="button"
              class="btn-primary inline-flex justify-center py-2 px-2"
              @click.prevent="enterWizard(true)"
            >
              {{ $t("workflow.configure-gitops") }}
              <heroicons-outline:chevron-right class="ml-1 w-5 h-5" />
            </button>
          </div>
        </template>
      </template>
      <template v-else-if="project.workflowType == 'VCS'">
        <RepositoryPanel
          :project="project"
          :repository="repository"
          :allow-edit="allowEdit"
          @change-repository="enterWizard(false)"
        />
      </template>
    </template>
  </div>
</template>

<script lang="ts">
import { reactive, watchEffect, watch, defineComponent } from "vue";
import { computed, PropType } from "vue";
import RepositorySetupWizard from "./RepositorySetupWizard.vue";
import RepositoryPanel from "./RepositoryPanel.vue";
import { Project, ProjectWorkflowType, UNKNOWN_ID } from "../types";
import { useI18n } from "vue-i18n";
import { pushNotification, useRepositoryStore } from "@/store";

interface LocalState {
  workflowType: ProjectWorkflowType;
  showWizardForCreate: boolean;
  showWizardForChange: boolean;
}

export default defineComponent({
  name: "ProjectVersionControlPanel",
  components: {
    RepositorySetupWizard,
    RepositoryPanel,
  },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
    allowEdit: {
      default: true,
      type: Boolean,
    },
  },
  async setup(props) {
    const { t } = useI18n();

    const repositoryStore = useRepositoryStore();

    const state = reactive<LocalState>({
      workflowType: props.project.workflowType,
      showWizardForCreate: false,
      showWizardForChange: false,
    });

    const prepareRepository = () => {
      repositoryStore.fetchRepositoryByProjectId(props.project.id);
    };

    watchEffect(prepareRepository);

    watch(
      () => props.project,
      (cur) => {
        state.workflowType = cur.workflowType;
      }
    );

    const repository = computed(() => {
      return repositoryStore.getRepositoryByProjectId(props.project.id);
    });

    const enterWizard = (create: boolean) => {
      if (create) {
        state.showWizardForCreate = true;
      } else {
        state.showWizardForChange = true;
      }
    };

    const cancelWizard = () => {
      state.workflowType = "UI";
      state.showWizardForCreate = false;
      state.showWizardForChange = false;
    };

    const finishWizard = () => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: state.showWizardForCreate
          ? t("workflow.configure-gitops-success", {
              project: props.project.name,
            })
          : t("workflow.change-gitops-success", {
              project: props.project.name,
            }),
      });
      state.showWizardForCreate = false;
      state.showWizardForChange = false;
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
});
</script>
