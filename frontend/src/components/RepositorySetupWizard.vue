<template>
  <div>
    <div class="textinfolabel">
      <i18n-t keypath="repository.setup-wizard-guide">
        <template #guide>
          <a
            href="https://bytebase.com/docs/vcs-integration/enable-gitops-workflow?source=console"
            target="_blank"
            class="normal-link"
          >
            {{ $t("common.detailed-guide") }}</a
          >
        </template>
      </i18n-t>
    </div>

    <StepTab
      class="mt-4 mb-8"
      :current-index="state.currentStep"
      :step-list="stepList"
      :allow-next="allowNext"
      @update:current-index="tryChangeStep"
      @finish="tryFinishSetup"
      @cancel="cancel"
    >
      <template #0="{ next }">
        <RepositoryVCSProviderPanel
          @next="next()"
          @set-vcs="setVCS"
          @set-code="setCode"
        />
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
import type { PropType } from "vue";
import { reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { StepTab } from "@/components/v2";
import { PROJECT_V1_ROUTE_GITOPS } from "@/router/dashboard/projectV1";
import { useRepositoryV1Store } from "@/store";
import type { Project } from "@/types/proto/v1/project_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import type {
  ProjectGitOpsInfo,
  VCSProvider,
} from "@/types/proto/v1/vcs_provider_service";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";
import type { ExternalRepositoryInfo, ProjectRepositoryConfig } from "../types";

// Default file path template is to organize migration files from different environments under separate directories.
const DEFAULT_FILE_PATH_TEMPLATE =
  "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql";
// Default schema path template is co-locate with the corresponding db's migration files and use .(dot) to appear the first.
const DEFAULT_SCHEMA_PATH_TEMPLATE = "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql";

// For tenant mode projects, {{ENV_ID}} and {{DB_NAME}} is not supported.
const DEFAULT_TENANT_MODE_FILE_PATH_TEMPLATE =
  "{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql";
const DEFAULT_TENANT_MODE_SCHEMA_PATH_TEMPLATE = ".LATEST.sql";

const CHOOSE_PROVIDER_STEP = 0;
// const CHOOSE_REPOSITORY_STEP = 1;
const CONFIGURE_DEPLOY_STEP = 2;

interface LocalState {
  config: ProjectRepositoryConfig;
  currentStep: number;
  showFeatureModal: boolean;
  processing: boolean;
}

const props = defineProps({
  // If false, then we intend to change the existing linked repository intead of just linking a new repository.
  create: {
    type: Boolean,
    default: false,
  },
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "finish"): void;
}>();

const { t } = useI18n();

const router = useRouter();
const repositoryV1Store = useRepositoryV1Store();

const stepList = [
  { title: t("repository.choose-git-provider"), hideNext: true },
  { title: t("repository.select-repository"), hideNext: true },
  { title: t("repository.configure-deploy") },
];

const isTenantProject = computed(() => {
  return props.project.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const state = reactive<LocalState>({
  config: {
    vcs: {} as VCSProvider,
    code: "",
    repositoryInfo: {
      externalId: "",
      name: "",
      fullPath: "",
      webUrl: "",
    },
    repositoryConfig: {
      baseDirectory: "bytebase",
      branchFilter: "main",
      filePathTemplate: isTenantProject.value
        ? DEFAULT_TENANT_MODE_FILE_PATH_TEMPLATE
        : DEFAULT_FILE_PATH_TEMPLATE,
      schemaPathTemplate: isTenantProject.value
        ? DEFAULT_TENANT_MODE_SCHEMA_PATH_TEMPLATE
        : DEFAULT_SCHEMA_PATH_TEMPLATE,
    },
  },
  currentStep: CHOOSE_PROVIDER_STEP,
  showFeatureModal: false,
  processing: false,
});

const allowNext = computed((): boolean => {
  if (state.currentStep == CONFIGURE_DEPLOY_STEP) {
    return (
      !isEmpty(state.config.repositoryConfig.branchFilter.trim()) &&
      !isEmpty(state.config.repositoryConfig.filePathTemplate.trim()) &&
      !state.processing
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

const tryFinishSetup = async () => {
  if (state.processing) {
    return;
  }
  state.processing = true;

  const createFunc = async () => {
    let externalId = state.config.repositoryInfo.externalId;
    if (
      state.config.vcs.type === VCSProvider_Type.GITHUB ||
      state.config.vcs.type === VCSProvider_Type.BITBUCKET
    ) {
      externalId = state.config.repositoryInfo.fullPath;
    }

    const repositoryCreate: Partial<ProjectGitOpsInfo> = {
      vcs: state.config.vcs.name,
      title: state.config.repositoryInfo.name,
      fullPath: state.config.repositoryInfo.fullPath,
      webUrl: state.config.repositoryInfo.webUrl,
      branchFilter: state.config.repositoryConfig.branchFilter,
      baseDirectory: state.config.repositoryConfig.baseDirectory,
      filePathTemplate: state.config.repositoryConfig.filePathTemplate,
      schemaPathTemplate: state.config.repositoryConfig.schemaPathTemplate,
      externalId: externalId,
    };
    await repositoryV1Store.upsertRepository(
      props.project.name,
      repositoryCreate
    );

    emit("finish");
  };

  try {
    if (!props.create) {
      // It's simple to implement change behavior as delete followed by create.
      // Though the delete can succeed while the create fails, this is rare, and
      // even it happens, user can still configure it again.
      await repositoryV1Store.deleteRepository(props.project.name);
    }
    await createFunc();
  } finally {
    state.processing = false;
  }
};

const cancel = () => {
  emit("cancel");
  router.push({
    name: PROJECT_V1_ROUTE_GITOPS,
  });
};

const setCode = (code: string) => {
  state.config.code = code;
};

const setVCS = (vcs: VCSProvider) => {
  state.config.vcs = vcs;
};

const setRepository = (repository: ExternalRepositoryInfo) => {
  state.config.repositoryInfo = repository;
};
</script>
