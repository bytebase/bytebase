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
    <BBStepTab
      class="mt-4 mb-8"
      :step-item-list="stepList"
      :allow-next="allowNext"
      @try-change-step="tryChangeStep"
      @try-finish="tryFinishSetup"
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
          @set-token="setToken"
          @set-repository="setRepository"
        />
      </template>
      <template #2>
        <RepositoryConfigPanel
          :config="state.config"
          :project="project"
          @change-schema-change-type="setSchemaChangeType"
        />
      </template>
    </BBStepTab>
    <BBModal
      v-if="state.showSetupSQLReviewCIModal"
      class="relative overflow-hidden"
      :title="$t('repository.sql-review-ci-setup')"
      @close="closeSetupSQLReviewModal"
    >
      <div class="space-y-4 max-w-[32rem]">
        <div class="whitespace-pre-wrap">
          {{
            $t("repository.sql-review-ci-setup-modal", {
              pr:
                state.config.vcs.type === ExternalVersionControl_Type.GITLAB
                  ? $t("repository.merge-request")
                  : $t("repository.pull-request"),
            })
          }}
        </div>

        <div class="flex justify-end pt-4 gap-x-2">
          <a
            class="btn-primary items-center space-x-2 mx-2 my-2"
            :href="state.sqlReviewCIPullRequestURL"
            target="_blank"
          >
            {{
              $t("repository.sql-review-ci-setup-pr", {
                pr:
                  state.config.vcs.type === ExternalVersionControl_Type.GITLAB
                    ? $t("repository.merge-request")
                    : $t("repository.pull-request"),
              })
            }}
          </a>
        </div>
      </div>
    </BBModal>
    <BBAlert
      v-if="state.showSetupSQLReviewCIFailureModal"
      :style="'CRITICAL'"
      :ok-text="$t('common.retry')"
      :title="$t('repository.sql-review-ci-setup-failed')"
      @ok="
        () => {
          state.showSetupSQLReviewCIFailureModal = false;
          createSQLReviewCI();
        }
      "
      @cancel="
        () => {
          state.showSetupSQLReviewCIFailureModal = false;
          $emit('finish');
        }
      "
    >
    </BBAlert>
    <BBModal
      v-if="state.showLoadingSQLReviewPRModal"
      class="relative overflow-hidden"
      :show-close="false"
      :esc-closable="false"
      :title="$t('repository.sql-review-ci-setup')"
    >
      <div
        class="whitespace-pre-wrap max-w-[32rem] flex justify-start items-start gap-x-2"
      >
        <BBSpin class="mt-1" />
        {{
          $t("repository.sql-review-ci-loading-modal", {
            pr:
              state.config.vcs.type === ExternalVersionControl_Type.GITLAB
                ? $t("repository.merge-request")
                : $t("repository.pull-request"),
          })
        }}
      </div>
    </BBModal>
    <FeatureModal
      v-if="state.showFeatureModal"
      feature="bb.feature.vcs-sql-review"
      @cancel="state.showFeatureModal = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed, PropType } from "vue";
import { useRouter } from "vue-router";
import isEmpty from "lodash-es/isEmpty";
import { cloneDeep } from "lodash-es";
import { BBStepTabItem } from "../bbkit/types";
import {
  ExternalRepositoryInfo,
  OAuthToken,
  ProjectRepositoryConfig,
} from "../types";
import { projectSlugV1 } from "../utils";
import { useI18n } from "vue-i18n";
import { useRepositoryV1Store, hasFeature, useProjectV1Store } from "@/store";
import { getVCSUid } from "@/store/modules/v1/common";
import {
  Project,
  Workflow,
  TenantMode,
  SchemaChange,
} from "@/types/proto/v1/project_service";
import {
  ProjectGitOpsInfo,
  ExternalVersionControl,
  ExternalVersionControl_Type,
} from "@/types/proto/v1/externalvs_service";

// Default file path template is to organize migration files from different environments under separate directories.
const DEFAULT_FILE_PATH_TEMPLATE =
  "{{ENV_ID}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql";
// Default schema path template is co-locate with the corresponding db's migration files and use .(dot) to appear the first.
const DEFAULT_SCHEMA_PATH_TEMPLATE = "{{ENV_ID}}/.{{DB_NAME}}##LATEST.sql";
// Default sheet path tempalte is to organize script files for SQL Editor.
const DEFAULT_SHEET_PATH_TEMPLATE =
  "script/{{ENV_ID}}##{{DB_NAME}}##{{NAME}}.sql";

// For tenant mode projects, {{ENV_ID}} and {{DB_NAME}} is not supported.
const DEFAULT_TENANT_MODE_FILE_PATH_TEMPLATE =
  "{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql";
const DEFAULT_TENANT_MODE_SCHEMA_PATH_TEMPLATE = ".LATEST.sql";
const DEFAULT_TENANT_MODE_SHEET_PATH_TEMPLATE = "script/{{NAME}}.sql";

const CHOOSE_PROVIDER_STEP = 0;
// const CHOOSE_REPOSITORY_STEP = 1;
const CONFIGURE_DEPLOY_STEP = 2;

interface LocalState {
  config: ProjectRepositoryConfig;
  currentStep: number;
  showFeatureModal: boolean;
  showSetupSQLReviewCIModal: boolean;
  showSetupSQLReviewCIFailureModal: boolean;
  showLoadingSQLReviewPRModal: boolean;
  sqlReviewCIPullRequestURL: string;
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

const stepList: BBStepTabItem[] = [
  { title: t("repository.choose-git-provider"), hideNext: true },
  { title: t("repository.select-repository"), hideNext: true },
  { title: t("repository.configure-deploy") },
];

const isTenantProject = computed(() => {
  return props.project.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const state = reactive<LocalState>({
  config: {
    vcs: {} as ExternalVersionControl,
    code: "",
    token: {
      accessToken: "",
      expiresTs: 0,
      refreshToken: "",
    },
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
      sheetPathTemplate: isTenantProject.value
        ? DEFAULT_TENANT_MODE_SHEET_PATH_TEMPLATE
        : DEFAULT_SHEET_PATH_TEMPLATE,
      enableSQLReviewCI: false,
    },
    schemaChangeType: props.project.schemaChange,
  },
  currentStep: CHOOSE_PROVIDER_STEP,
  showFeatureModal: false,
  showSetupSQLReviewCIModal: false,
  showSetupSQLReviewCIFailureModal: false,
  showLoadingSQLReviewPRModal: false,
  sqlReviewCIPullRequestURL: "",
});

const allowNext = computed((): boolean => {
  if (state.currentStep == CONFIGURE_DEPLOY_STEP) {
    return (
      !isEmpty(state.config.repositoryConfig.branchFilter.trim()) &&
      !isEmpty(state.config.repositoryConfig.filePathTemplate.trim())
    );
  }
  return true;
});

const tryChangeStep = (
  oldStep: number,
  newStep: number,
  allowChangeCallback: () => void
) => {
  state.currentStep = newStep;
  allowChangeCallback();
};

const createSQLReviewCI = async () => {
  state.showLoadingSQLReviewPRModal = true;

  try {
    const pullRequestURL = await repositoryV1Store.setupSQLReviewCI(
      props.project.name
    );
    state.sqlReviewCIPullRequestURL = pullRequestURL;
    state.showSetupSQLReviewCIModal = true;
    window.open(pullRequestURL, "_blank");
  } catch {
    state.showSetupSQLReviewCIFailureModal = true;
  } finally {
    state.showLoadingSQLReviewPRModal = false;
  }
};

const tryFinishSetup = (allowFinishCallback: () => void) => {
  if (
    state.config.repositoryConfig.enableSQLReviewCI &&
    !hasFeature("bb.feature.vcs-sql-review")
  ) {
    state.showFeatureModal = true;
    return;
  }

  const createFunc = async () => {
    let externalId = state.config.repositoryInfo.externalId;
    if (
      state.config.vcs.type === ExternalVersionControl_Type.GITHUB ||
      state.config.vcs.type === ExternalVersionControl_Type.BITBUCKET
    ) {
      externalId = state.config.repositoryInfo.fullPath;
    }

    const projectPatch = cloneDeep(props.project);
    projectPatch.workflow = Workflow.VCS;
    const updateMask = ["workflow"];
    // Update project schemaChangeType field.
    if (state.config.schemaChangeType !== props.project.schemaChange) {
      updateMask.push("schema_change");
      projectPatch.schemaChange = state.config.schemaChangeType;
    }
    await useProjectV1Store().updateProject(projectPatch, updateMask);

    const repositoryCreate: Partial<ProjectGitOpsInfo> = {
      vcsUid: `${getVCSUid(state.config.vcs.name)}`,
      title: state.config.repositoryInfo.name,
      fullPath: state.config.repositoryInfo.fullPath,
      webUrl: state.config.repositoryInfo.webUrl,
      branchFilter: state.config.repositoryConfig.branchFilter,
      baseDirectory: state.config.repositoryConfig.baseDirectory,
      filePathTemplate: state.config.repositoryConfig.filePathTemplate,
      schemaPathTemplate: state.config.repositoryConfig.schemaPathTemplate,
      sheetPathTemplate: state.config.repositoryConfig.sheetPathTemplate,
      externalId: externalId,
      accessToken: state.config.token.accessToken,
      expiresTime: new Date(state.config.token.expiresTs * 1000),
      refreshToken: state.config.token.refreshToken,
      enableSqlReviewCi: false,
    };
    await repositoryV1Store.upsertRepository(
      props.project.name,
      repositoryCreate
    );

    if (state.config.repositoryConfig.enableSQLReviewCI) {
      createSQLReviewCI();
    } else {
      emit("finish");
    }

    allowFinishCallback();
  };

  if (props.create) {
    createFunc();
  } else {
    // It's simple to implement change behavior as delete followed by create.
    // Though the delete can succeed while the create fails, this is rare, and
    // even it happens, user can still configure it again.
    repositoryV1Store.deleteRepository(props.project.name).then(() => {
      createFunc();
    });
  }
};

const closeSetupSQLReviewModal = () => {
  state.showSetupSQLReviewCIModal = false;
  emit("finish");
};

const cancel = () => {
  emit("cancel");
  router.push({
    name: "workspace.project.detail",
    params: {
      projectSlug: projectSlugV1(props.project),
    },
    hash: "#gitops",
  });
};

const setCode = (code: string) => {
  state.config.code = code;
};

const setToken = (token: OAuthToken) => {
  state.config.token = token;
};

const setVCS = (vcs: ExternalVersionControl) => {
  state.config.vcs = vcs;
};

const setRepository = (repository: ExternalRepositoryInfo) => {
  state.config.repositoryInfo = repository;
};

const setSchemaChangeType = (schemaChange: SchemaChange) => {
  state.config.schemaChangeType = schemaChange;
};
</script>
