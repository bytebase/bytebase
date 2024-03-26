<template>
  <div class="flex justify-between">
    <div class="text-lg leading-6 font-medium text-main">
      <i18n-t keypath="repository.gitops-status">
        <template #status>
          <span class="text-success"> {{ $t("common.enabled") }} </span>
        </template>
      </i18n-t>
    </div>
    <TroubleshootLink
      url="https://www.bytebase.com/docs/vcs-integration/troubleshoot/?source=console"
    />
  </div>
  <div class="mt-2 textinfolabel">
    <i18n-t keypath="repository.gitops-description-file-path">
      <template #fullPath>
        <a class="normal-link" :href="repository.webUrl" target="_blank">
          {{ repositoryFormattedFullPath }}
        </a>
      </template>
    </i18n-t>
    <span>&nbsp;</span>
    <i18n-t keypath="repository.gitops-description-branch">
      <template #branch>
        <span class="font-medium text-main">
          <template v-if="state.repositoryConfig.branch">
            {{ state.repositoryConfig.branch }}
          </template>
          <template v-else>
            {{ $t("common.default") }}
          </template>
        </span>
      </template>
    </i18n-t>
  </div>
  <RepositoryForm
    class="mt-4"
    :allow-edit="allowEdit"
    :vcs-type="vcs.type"
    :vcs-name="vcs.title"
    :repository-info="repositoryInfo"
    :repository-config="state.repositoryConfig"
    :project="project"
    @change-repository="$emit('change-repository')"
  />
  <div v-if="allowEdit" class="mt-4 pt-4 flex border-t justify-between">
    <BBButtonConfirm
      :style="'RESTORE'"
      :button-text="$t('repository.restore-to-ui-workflow')"
      :require-confirm="true"
      :ok-text="$t('common.restore')"
      :confirm-title="$t('repository.restore-to-ui-workflow') + '?'"
      :confirm-description="$t('repository.restore-ui-workflow-description')"
      @confirm="() => restoreToUIWorkflowType(true)"
    />
    <div class="ml-3">
      <NButton
        type="primary"
        :disabled="!allowUpdate"
        @click.prevent="doUpdate"
      >
        {{ $t("common.update") }}
      </NButton>
    </div>
  </div>
  <FeatureModal
    feature="bb.feature.vcs-sql-review"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import isEmpty from "lodash-es/isEmpty";
import type { PropType } from "vue";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useProjectV1Store,
  useRepositoryV1Store,
} from "@/store";
import type { Project } from "@/types/proto/v1/project_service";
import type {
  ProjectGitOpsInfo,
  VCSProvider,
} from "@/types/proto/v1/vcs_provider_service";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";
import type { ExternalRepositoryInfo, RepositoryConfig } from "../types";

interface LocalState {
  repositoryConfig: RepositoryConfig;
  showFeatureModal: boolean;
  processing: boolean;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  repository: {
    required: true,
    type: Object as PropType<ProjectGitOpsInfo>,
  },
  vcs: {
    required: true,
    type: Object as PropType<VCSProvider>,
  },
  allowEdit: {
    default: true,
    type: Boolean,
  },
});

const emit = defineEmits<{
  (event: "change-repository"): void;
  (event: "restore"): void;
}>();

const { t } = useI18n();
const repositoryV1Store = useRepositoryV1Store();
const projectV1Store = useProjectV1Store();
const state = reactive<LocalState>({
  repositoryConfig: {
    baseDirectory: props.repository.baseDirectory,
    branch: props.repository.branch,
  },
  showFeatureModal: false,
  processing: false,
});

watch(
  () => props.repository,
  (cur) => {
    state.repositoryConfig = {
      baseDirectory: cur.baseDirectory,
      branch: cur.branch,
    };
  }
);

const repositoryFormattedFullPath = computed(() => {
  const fullPath = props.repository.fullPath;
  if (props.vcs.type !== VCSProvider_Type.AZURE_DEVOPS) {
    return fullPath;
  }
  if (!fullPath.includes("@dev.azure.com")) {
    return fullPath;
  }
  return `https://dev.azure.com${fullPath.split("@dev.azure.com")[1]}`;
});

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
    !state.processing &&
    !isEmpty(state.repositoryConfig.branch) &&
    (props.repository.branch !== state.repositoryConfig.branch ||
      props.repository.baseDirectory !== state.repositoryConfig.baseDirectory)
  );
});

const restoreToUIWorkflowType = async (checkSQLReviewCI: boolean) => {
  if (state.processing) {
    return;
  }
  state.processing = true;

  try {
    await repositoryV1Store.deleteRepository(props.project.name);
    await projectV1Store.fetchProjectByName(props.project.name);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("repository.restore-ui-workflow-success"),
    });

    emit("restore");
  } finally {
    state.processing = false;
  }
};

const doUpdate = async () => {
  if (state.processing) {
    return;
  }
  state.processing = true;

  const repositoryPatch: Partial<ProjectGitOpsInfo> = {};

  repositoryPatch.vcs = props.vcs.name;

  if (props.repository.branch != state.repositoryConfig.branch) {
    repositoryPatch.branch = state.repositoryConfig.branch;
  }
  if (props.repository.baseDirectory != state.repositoryConfig.baseDirectory) {
    repositoryPatch.baseDirectory = state.repositoryConfig.baseDirectory;
  }

  try {
    await repositoryV1Store.upsertRepository(
      props.project.name,
      repositoryPatch
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("repository.update-gitops-config-success"),
    });
  } finally {
    state.processing = false;
  }
};
</script>
