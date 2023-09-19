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
      <template v-if="project.workflow === Workflow.UI">
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
              :value="Workflow.UI"
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
              id="workflow-gitops"
              v-model="state.workflowType"
              name="GitOps workflow"
              tabindex="-1"
              type="radio"
              class="text-accent disabled:text-accent-disabled focus:ring-accent"
              :value="Workflow.VCS"
              :disabled="!allowEdit"
            />
            <div class="-mt-1">
              <label for="workflow-gitops" class="textlabel">{{
                $t("workflow.gitops-workflow")
              }}</label>
              <div class="mt-1 textinfolabel">
                {{ $t("workflow.gitops-workflow-description") }}
              </div>
            </div>
          </div>
        </div>
        <template v-if="allowEdit && state.workflowType == Workflow.VCS">
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
      <template v-else-if="project.workflow === Workflow.VCS && repository">
        <RepositoryPanel
          :project="project"
          :vcs="vcs"
          :repository="repository"
          :allow-edit="allowEdit"
          @change-repository="enterWizard(false)"
          @restore="cancelWizard"
        />
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { reactive, watchEffect, watch } from "vue";
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useRepositoryV1Store, useVCSV1Store } from "@/store";
import { ExternalVersionControl } from "@/types/proto/v1/externalvs_service";
import { Project, Workflow } from "@/types/proto/v1/project_service";

interface LocalState {
  workflowType: Workflow;
  showWizardForCreate: boolean;
  showWizardForChange: boolean;
}

const props = defineProps({
  allowEdit: {
    default: true,
    type: Boolean,
  },
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const { t } = useI18n();

const repositoryV1Store = useRepositoryV1Store();
const vcsV1Store = useVCSV1Store();

const state = reactive<LocalState>({
  workflowType: props.project.workflow,
  showWizardForCreate: false,
  showWizardForChange: false,
});

watchEffect(async () => {
  const repo = await repositoryV1Store.getOrFetchRepositoryByProject(
    props.project.name,
    true /* silent */
  );
  if (repo) {
    await vcsV1Store.fetchVCSByUid(repo.vcsUid);
  }
});

watch(
  () => props.project.workflow,
  (type) => {
    state.workflowType = type;
  }
);

const repository = computed(() => {
  return repositoryV1Store.getRepositoryByProject(props.project.name);
});

const vcs = computed(() => {
  return (
    vcsV1Store.getVCSByUid(repository.value?.vcsUid ?? "") ??
    ({} as ExternalVersionControl)
  );
});

const enterWizard = (create: boolean) => {
  if (create) {
    state.showWizardForCreate = true;
  } else {
    state.showWizardForChange = true;
  }
};

const cancelWizard = () => {
  state.workflowType = Workflow.UI;
  state.showWizardForCreate = false;
  state.showWizardForChange = false;
};

const finishWizard = () => {
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: state.showWizardForCreate
      ? t("workflow.configure-gitops-success", {
          project: props.project.title,
        })
      : t("workflow.change-gitops-success", {
          project: props.project.title,
        }),
  });
  state.workflowType = Workflow.VCS;
  state.showWizardForCreate = false;
  state.showWizardForChange = false;
};
</script>
