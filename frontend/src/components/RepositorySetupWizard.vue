<template>
  <div class="w-full h-full overflow-hidden flex flex-col space-y-4">
    <BBAttention type="info">
      <div class="textinfolabel">
        <i18n-t keypath="repository.setup-wizard-guide">
          <template #guide>
            <a
              href="https://bytebase.com/docs/vcs-integration/add-gitops-connector?source=console"
              target="_blank"
              class="normal-link"
            >
              {{ $t("common.detailed-guide") }}
            </a>
          </template>
        </i18n-t>
      </div>
    </BBAttention>

    <StepTab
      class="pt-4 flex-1 overflow-hidden flex flex-col"
      pane-class="flex-1 overflow-y-auto"
      :current-index="state.currentStep"
      :step-list="stepList"
      :allow-next="allowNext"
      @update:current-index="tryChangeStep"
      @finish="tryFinishSetup"
      @cancel="cancel"
    >
      <template #0="{ next }">
        <RepositoryVCSProviderPanel @next="next()" @set-vcs="setVCS" />
      </template>
      <template #1="{ next }">
        <RepositorySelectionPanel
          :config="state.config"
          @next="next()"
          @set-repository="setRepository"
        />
      </template>
      <template #2>
        <RepositoryConfigPanel :config="state.config" :project="project" />
      </template>
    </StepTab>
  </div>
</template>

<script lang="ts" setup>
import isEmpty from "lodash-es/isEmpty";
import { reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention } from "@/bbkit";
import { StepTab } from "@/components/v2";
import { pushNotification, useVCSConnectorStore } from "@/store";
import type { ComposedProject } from "@/types";
import { VCSType } from "@/types/proto/v1/common";
import { VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import { VCSRepository } from "@/types/proto/v1/vcs_provider_service";
import { hasProjectPermissionV2 } from "@/utils";
import type { ProjectRepositoryConfig } from "../types";
import RepositoryConfigPanel from "./RepositoryConfigPanel.vue";
import RepositorySelectionPanel from "./RepositorySelectionPanel.vue";
import RepositoryVCSProviderPanel from "./RepositoryVCSProviderPanel.vue";

const CHOOSE_PROVIDER_STEP = 0;
// const CHOOSE_REPOSITORY_STEP = 1;
const CONFIGURE_DEPLOY_STEP = 2;

interface LocalState {
  config: ProjectRepositoryConfig;
  currentStep: number;
  showFeatureModal: boolean;
  processing: boolean;
}

const props = defineProps<{
  project: ComposedProject;
}>();

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "finish"): void;
}>();

const { t } = useI18n();
const vcsConnectorStore = useVCSConnectorStore();

const stepList = [
  { title: t("repository.choose-git-provider"), hideNext: true },
  { title: t("repository.select-repository"), hideNext: true },
  { title: t("repository.configure-deploy") },
];

const state = reactive<LocalState>({
  config: {
    vcs: VCSProvider.fromPartial({}),
    repositoryInfo: VCSRepository.fromPartial({}),
    repositoryConfig: {
      baseDirectory: "/bytebase",
      branch: "main",
      resourceId: "",
      databaseGroup: "",
    },
  },
  currentStep: CHOOSE_PROVIDER_STEP,
  showFeatureModal: false,
  processing: false,
});

const hasPermission = computed(() => {
  return hasProjectPermissionV2(props.project, "bb.vcsConnectors.create");
});

const allowNext = computed((): boolean => {
  if (state.currentStep == CONFIGURE_DEPLOY_STEP) {
    return (
      !isEmpty(state.config.repositoryConfig.branch.trim()) &&
      !isEmpty(state.config.repositoryConfig.resourceId.trim()) &&
      !state.processing &&
      hasPermission.value
    );
  }
  return true;
});

const tryChangeStep = (nextStepIndex: number) => {
  if (state.processing) {
    return;
  }
  state.currentStep = nextStepIndex;
};

const validBaseDirectory = computed(() => {
  let directory = state.config.repositoryConfig.baseDirectory;
  if (directory.endsWith("/")) {
    directory = directory.slice(0, directory.length - 1);
  }
  return directory || "/";
});

const tryFinishSetup = async () => {
  if (state.processing) {
    return;
  }
  state.processing = true;

  const createFunc = async () => {
    let externalId = state.config.repositoryInfo.id;
    if (
      state.config.vcs.type === VCSType.GITHUB ||
      state.config.vcs.type === VCSType.BITBUCKET
    ) {
      externalId = state.config.repositoryInfo.fullPath;
    }

    await vcsConnectorStore.createConnector(
      props.project.name,
      state.config.repositoryConfig.resourceId,
      {
        title: state.config.repositoryInfo.title,
        externalId,
        vcsProvider: state.config.vcs.name,
        baseDirectory: validBaseDirectory.value,
        branch: state.config.repositoryConfig.branch.trim(),
        databaseGroup: state.config.repositoryConfig.databaseGroup,
        fullPath: state.config.repositoryInfo.fullPath,
        webUrl: state.config.repositoryInfo.webUrl,
      }
    );

    emit("finish");
  };

  try {
    await createFunc();
  } catch (error: any) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Create connector error occurred`,
      description: error.details,
    });
  } finally {
    state.processing = false;
  }
};

const cancel = () => {
  if (state.processing) {
    return;
  }
  emit("cancel");
};

const setVCS = (vcs: VCSProvider) => {
  state.config.vcs = vcs;
};

const setRepository = (repository: VCSRepository) => {
  state.config.repositoryInfo = repository;
};
</script>
