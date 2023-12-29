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
        <div class="mt-6">
          <NRadioGroup
            v-model:value="state.workflowType"
            :disabled="!allowEdit"
            class="flex flex-col space-y-4"
          >
            <NRadio
              v-for="workflow in workflowList"
              :key="workflow.value"
              :value="workflow.value"
            >
              <div>
                <span class="textlabel">
                  {{ workflow.title }}
                </span>
                <div class="mt-1 textinfolabel">
                  {{ workflow.description }}
                </div>
              </div>
            </NRadio>
          </NRadioGroup>
        </div>
        <template v-if="allowEdit && state.workflowType == Workflow.VCS">
          <div class="mt-4 flex items-center justify-end">
            <NButton type="primary" @click.prevent="enterWizard(true)">
              {{ $t("workflow.configure-gitops") }}
              <template #icon>
                <heroicons-outline:chevron-right class="w-5 h-5" />
              </template>
            </NButton>
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
import { NRadio, NRadioGroup } from "naive-ui";
import { reactive, watch } from "vue";
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

const workflowList = computed(() => [
  {
    value: Workflow.UI,
    title: t("workflow.ui-workflow"),
    description: t("workflow.ui-workflow-description"),
  },
  {
    value: Workflow.VCS,
    title: t("workflow.gitops-workflow"),
    description: t("workflow.gitops-workflow-description"),
  },
]);

watch(
  () => [props.project.name],
  async () => {
    const repo = await repositoryV1Store.getOrFetchRepositoryByProject(
      props.project.name,
      true /* silent */
    );
    if (repo) {
      await vcsV1Store.fetchVCSByUid(repo.vcsUid);
    }
  },
  {
    immediate: true,
  }
);

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
