<template>
  <div>
    <div class="textinfolabel">
      <i18n-t keypath="repository.setup-wizard-guide">
        <template #guide>
          <a
            href="https://bytebase.com/docs/vcs-integration/enable-version-control-workflow?source=console"
            target="_blank"
            class="normal-link"
          >
            {{ $t("common.detailed-guide") }}</a
          >
        </template>
      </i18n-t>
    </div>
    <BBStepTab
      class="pt-4"
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
          @change-schema-change-type="
            (type) => (state.config.schemaChangeType = type)
          "
        />
      </template>
    </BBStepTab>
  </div>
</template>

<script lang="ts">
import { reactive, computed, PropType, defineComponent } from "vue";
import { useRouter } from "vue-router";
import isEmpty from "lodash-es/isEmpty";
import { BBStepTabItem } from "../bbkit/types";
import RepositoryVCSProviderPanel from "./RepositoryVCSProviderPanel.vue";
import RepositorySelectionPanel from "./RepositorySelectionPanel.vue";
import RepositoryConfigPanel from "./RepositoryConfigPanel.vue";
import {
  ExternalRepositoryInfo,
  OAuthToken,
  Project,
  ProjectPatch,
  ProjectRepositoryConfig,
  RepositoryCreate,
  unknown,
  VCS,
} from "../types";
import { projectSlug } from "../utils";
import { useI18n } from "vue-i18n";
import { useProjectStore, useRepositoryStore } from "@/store";

// Default file path template is to organize migration files from different environments under separate directories.
const DEFAULT_FILE_PATH_TEMPLATE =
  "{{ENV_NAME}}/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql";
// Default schema path template is co-locate with the corresponding db's migration files and use .(dot) to appear the first.
const DEFAULT_SCHEMA_PATH_TEMPLATE = "{{ENV_NAME}}/.{{DB_NAME}}__LATEST.sql";
// Default sheet path tempalte is to organize script files for SQL Editor.
const DEFAULT_SHEET_PATH_TEMPLATE =
  "script/{{ENV_NAME}}__{{DB_NAME}}__{{NAME}}.sql";

// For tenant mode projects, {{ENV_NAME}} is not supported.
const DEFAULT_TENANT_MODE_FILE_PATH_TEMPLATE =
  "{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql";
const DEFAULT_TENANT_MODE_SCHEMA_PATH_TEMPLATE = ".{{DB_NAME}}__LATEST.sql";
const DEFAULT_TENANT_MODE_SHEET_PATH_TEMPLATE = "script/{{NAME}}.sql";

const CHOOSE_PROVIDER_STEP = 0;
// const CHOOSE_REPOSITORY_STEP = 1;
const CONFIGURE_DEPLOY_STEP = 2;

interface LocalState {
  config: ProjectRepositoryConfig;
  currentStep: number;
}

export default defineComponent({
  name: "RepositorySetupWizard",
  components: {
    RepositoryVCSProviderPanel,
    RepositorySelectionPanel,
    RepositoryConfigPanel,
  },
  props: {
    // If false, then we intend to change the existing linked repository intead of just linking a new repository.
    create: {
      type: Boolean,
      default: false,
    },
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  emits: ["cancel", "finish"],
  setup(props, { emit }) {
    const { t } = useI18n();

    const router = useRouter();
    const repositoryStore = useRepositoryStore();

    const stepList: BBStepTabItem[] = [
      { title: t("repository.choose-git-provider"), hideNext: true },
      { title: t("repository.select-repository"), hideNext: true },
      { title: t("repository.configure-deploy") },
    ];

    const isTenantProject = computed(() => {
      return props.project.tenantMode === "TENANT";
    });

    const state = reactive<LocalState>({
      config: {
        vcs: unknown("VCS") as VCS,
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
        },
        schemaChangeType: props.project.schemaChangeType,
      },
      currentStep: CHOOSE_PROVIDER_STEP,
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

    const tryFinishSetup = (allowFinishCallback: () => void) => {
      const createFunc = async () => {
        let externalId = state.config.repositoryInfo.externalId;
        if (state.config.vcs.type == "GITHUB_COM") {
          externalId = state.config.repositoryInfo.fullPath;
        }

        // Update project schemaChangeType field firstly.
        if (state.config.schemaChangeType !== props.project.schemaChangeType) {
          const projectPatch: ProjectPatch = {
            schemaChangeType: state.config.schemaChangeType,
          };
          await useProjectStore().patchProject({
            projectId: props.project.id,
            projectPatch,
          });
        }

        const repositoryCreate: RepositoryCreate = {
          vcsId: state.config.vcs.id,
          name: state.config.repositoryInfo.name,
          fullPath: state.config.repositoryInfo.fullPath,
          webUrl: state.config.repositoryInfo.webUrl,
          branchFilter: state.config.repositoryConfig.branchFilter,
          baseDirectory: state.config.repositoryConfig.baseDirectory,
          filePathTemplate: state.config.repositoryConfig.filePathTemplate,
          schemaPathTemplate: state.config.repositoryConfig.schemaPathTemplate,
          sheetPathTemplate: state.config.repositoryConfig.sheetPathTemplate,
          externalId: externalId,
          accessToken: state.config.token.accessToken,
          expiresTs: state.config.token.expiresTs,
          refreshToken: state.config.token.refreshToken,
        };
        await repositoryStore.createRepository({
          projectId: props.project.id,
          repositoryCreate,
        });
        allowFinishCallback();
        emit("finish");
      };

      if (props.create) {
        createFunc();
      } else {
        // It's simple to implement change behavior as delete followed by create.
        // Though the delete can succeed while the create fails, this is rare, and
        // even it happens, user can still configure it again.
        repositoryStore
          .deleteRepositoryByProjectId(props.project.id)
          .then(() => {
            createFunc();
          });
      }
    };

    const cancel = () => {
      emit("cancel");
      router.push({
        name: "workspace.project.detail",
        params: {
          projectSlug: projectSlug(props.project),
        },
        hash: "#version-control",
      });
    };

    const setCode = (code: string) => {
      state.config.code = code;
    };

    const setToken = (token: OAuthToken) => {
      state.config.token = token;
    };

    const setVCS = (vcs: VCS) => {
      state.config.vcs = vcs;
    };

    const setRepository = (repository: ExternalRepositoryInfo) => {
      state.config.repositoryInfo = repository;
    };

    return {
      state,
      stepList,
      allowNext,
      tryChangeStep,
      tryFinishSetup,
      cancel,
      setCode,
      setVCS,
      setToken,
      setRepository,
    };
  },
});
</script>
