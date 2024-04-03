<template>
  <div class="mt-2 textinfolabel">
    <i18n-t keypath="repository.gitops-description-file-path">
      <template #fullPath>
        <a class="normal-link" :href="vcsConnector.webUrl" target="_blank">
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
    <span>&nbsp;</span>
    <TroubleshootLink
      url="https://www.bytebase.com/docs/vcs-integration/troubleshoot/?source=console"
    />
  </div>
  <RepositoryForm
    class="mt-4"
    :allow-edit="allowEdit"
    :vcs-type="vcsProvider.type"
    :vcs-name="vcsProvider.title"
    :repository-info="repositoryInfo"
    :repository-config="state.repositoryConfig"
    :project="project"
  />
  <div
    v-if="allowEdit || allowDelete"
    class="mt-6 pt-4 flex border-t justify-between"
  >
    <BBButtonConfirm
      v-if="allowDelete"
      :style="'DELETE'"
      :button-text="$t('project.gitops-connector.delete')"
      :require-confirm="true"
      :ok-text="$t('common.delete')"
      :confirm-title="$t('project.gitops-connector.delete') + '?'"
      @confirm="deleteConnector"
    />
    <div v-if="allowEdit" class="ml-3 flex items-center space-x-3">
      <NButton
        v-if="allowUpdate"
        :disabled="state.processing"
        @click.prevent="discardChanges"
      >
        {{ $t("common.discard-changes") }}
      </NButton>
      <NButton
        type="primary"
        :disabled="!allowUpdate"
        :loading="state.processing"
        @click.prevent="doUpdate"
      >
        {{ $t("common.update") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import isEmpty from "lodash-es/isEmpty";
import { computed, reactive, watch } from "vue";
import { useVCSProviderStore, useVCSConnectorStore } from "@/store";
import { getVCSConnectorId } from "@/store/modules/v1/common";
import { VCSType } from "@/types/proto/v1/common";
import type { Project } from "@/types/proto/v1/project_service";
import { VCSConnector } from "@/types/proto/v1/vcs_connector_service";
import { VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import type { VCSRepository } from "@/types/proto/v1/vcs_provider_service";
import type { RepositoryConfig } from "../types";

interface LocalState {
  repositoryConfig: RepositoryConfig;
  processing: boolean;
}

const props = defineProps<{
  project: Project;
  vcsConnector: VCSConnector;
  allowEdit: boolean;
  allowDelete: boolean;
}>();

const emit = defineEmits<{
  (event: "delete"): void;
  (event: "update"): void;
  (event: "cancel"): void;
}>();

const vcsV1Store = useVCSProviderStore();
const vcsConnectorStore = useVCSConnectorStore();

const initConfig = computed(
  (): RepositoryConfig => ({
    resourceId: getVCSConnectorId(props.vcsConnector.name).vcsConnectorId,
    baseDirectory: props.vcsConnector.baseDirectory,
    branch: props.vcsConnector.branch,
    databaseGroup: props.vcsConnector.databaseGroup,
  })
);

const state = reactive<LocalState>({
  repositoryConfig: { ...initConfig.value },
  processing: false,
});

const discardChanges = () => {
  state.repositoryConfig = { ...initConfig.value };
  state.processing = false;
};

watch(
  () => props.vcsConnector,
  () => {
    discardChanges();
  },
  { deep: true, immediate: true }
);

const vcsProvider = computed(
  () =>
    vcsV1Store.getVCSByName(props.vcsConnector.vcsProvider) ??
    VCSProvider.fromPartial({})
);

const repositoryFormattedFullPath = computed(() => {
  const fullPath = props.vcsConnector.fullPath;
  if (vcsProvider.value.type !== VCSType.AZURE_DEVOPS) {
    return fullPath;
  }
  if (!fullPath.includes("@dev.azure.com")) {
    return fullPath;
  }
  return `https://dev.azure.com${fullPath.split("@dev.azure.com")[1]}`;
});

const repositoryInfo = computed((): VCSRepository => {
  return {
    id: props.vcsConnector.externalId,
    title: props.vcsConnector.title,
    fullPath: props.vcsConnector.fullPath,
    webUrl: props.vcsConnector.webUrl,
  };
});

const allowUpdate = computed(() => {
  return (
    !isEmpty(state.repositoryConfig.branch) &&
    (props.vcsConnector.branch !== state.repositoryConfig.branch ||
      props.vcsConnector.baseDirectory !==
        state.repositoryConfig.baseDirectory ||
      props.vcsConnector.databaseGroup !== state.repositoryConfig.databaseGroup)
  );
});

const deleteConnector = async () => {
  if (state.processing) {
    return;
  }
  state.processing = true;

  try {
    await vcsConnectorStore.deleteConnector(props.vcsConnector.name);
    emit("delete");
  } finally {
    state.processing = false;
  }
};

const doUpdate = async () => {
  if (state.processing) {
    return;
  }
  state.processing = true;

  try {
    await vcsConnectorStore.updateConnector(
      VCSConnector.fromPartial({
        ...props.vcsConnector,
        branch: state.repositoryConfig.branch,
        baseDirectory: state.repositoryConfig.baseDirectory,
        databaseGroup: state.repositoryConfig.databaseGroup,
      }),
      ["branch", "base_directory", "database_group"]
    );
    emit("update");
  } finally {
    state.processing = false;
  }
};
</script>
