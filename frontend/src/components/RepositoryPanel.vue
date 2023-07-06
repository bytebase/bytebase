<template>
  <div class="text-lg leading-6 font-medium text-main">
    <i18n-t keypath="repository.gitops-status">
      <template #status>
        <span class="text-success"> {{ $t("common.enabled") }} </span>
      </template>
    </i18n-t>
  </div>
  <div class="mt-2 textinfolabel">
    <template v-if="isProjectSchemaChangeTypeDDL">
      <i18n-t keypath="repository.gitops-description-file-path">
        <template #fullPath>
          <a class="normal-link" :href="repository.webUrl" target="_blank">{{
            repository.fullPath
          }}</a>
        </template>
        <template #fullPathTemplate>
          <span class="font-medium text-main"
            >{{ state.repositoryConfig.baseDirectory }}/{{
              state.repositoryConfig.filePathTemplate
            }}</span
          >
        </template>
      </i18n-t>
      <span>&nbsp;</span>
      <i18n-t keypath="repository.gitops-description-branch">
        <template #branch>
          <span class="font-medium text-main">
            <template v-if="state.repositoryConfig.branchFilter">
              {{ state.repositoryConfig.branchFilter }}
            </template>
            <template v-else>
              {{ $t("common.default") }}
            </template>
          </span>
        </template>
      </i18n-t>
      <template v-if="state.repositoryConfig.schemaPathTemplate">
        <span>&nbsp;</span>
        <i18n-t keypath="repository.gitops-description-description-schema-path">
          <template #schemaPathTemplate>
            <span class="font-medium text-main">{{
              state.repositoryConfig.schemaPathTemplate
            }}</span>
          </template>
        </i18n-t>
      </template>
    </template>
    <template v-if="isProjectSchemaChangeTypeSDL">
      <i18n-t keypath="repository.gitops-description-sdl">
        <template #fullPath>
          <a class="normal-link" :href="repository.webUrl" target="_blank">
            {{ repository.fullPath }}
          </a>
        </template>
        <template #branch>
          <span class="font-medium text-main">
            <template v-if="state.repositoryConfig.branchFilter">
              {{ state.repositoryConfig.branchFilter }}
            </template>
            <template v-else>
              {{ $t("common.default") }}
            </template>
          </span>
        </template>
        <template #filePathTemplate>
          <span class="font-medium text-main">
            {{ state.repositoryConfig.baseDirectory }}/{{
              state.repositoryConfig.filePathTemplate
            }}
          </span>
        </template>
        <template #schemaPathTemplate>
          <span class="font-medium text-main">
            {{ state.repositoryConfig.schemaPathTemplate }}
          </span>
        </template>
      </i18n-t>
    </template>
  </div>
  <RepositoryForm
    class="mt-4"
    :allow-edit="allowEdit"
    :vcs-type="vcs.type"
    :vcs-name="vcs.title"
    :repository-info="repositoryInfo"
    :repository-config="state.repositoryConfig"
    :project="project"
    :schema-change-type="state.schemaChangeType"
    @change-schema-change-type="(type: SchemaChange) => (state.schemaChangeType = type)"
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
    <div>
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowUpdate"
        @click.prevent="doUpdate"
      >
        {{ $t("common.update") }}
      </button>
    </div>
  </div>
  <BBModal
    v-if="supportSQLReviewCI && state.showSetupSQLReviewCIModal"
    class="relative overflow-hidden"
    :title="$t('repository.sql-review-ci-setup')"
    @close="state.showSetupSQLReviewCIModal = false"
  >
    <div class="space-y-4 max-w-[32rem]">
      <div class="whitespace-pre-wrap">
        {{
          $t("repository.sql-review-ci-setup-modal", {
            pr:
              vcs.type === ExternalVersionControl_Type.GITLAB
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
                vcs.type === ExternalVersionControl_Type.GITLAB
                  ? $t("repository.merge-request")
                  : $t("repository.pull-request"),
            })
          }}
        </a>
      </div>
    </div>
  </BBModal>
  <BBAlert
    v-if="supportSQLReviewCI && state.showSetupSQLReviewCIFailureModal"
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
      }
    "
  >
  </BBAlert>
  <BBModal
    v-if="supportSQLReviewCI && state.showLoadingSQLReviewPRModal"
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
            vcs.type === ExternalVersionControl_Type.GITLAB
              ? $t("repository.merge-request")
              : $t("repository.pull-request"),
        })
      }}
    </div>
  </BBModal>
  <BBModal
    v-if="supportSQLReviewCI && state.showRestoreSQLReviewCIModal"
    class="relative overflow-hidden"
    :title="$t('repository.sql-review-ci-remove')"
    @close="onSQLReviewCIModalClose"
  >
    <div class="space-y-4 max-w-[32rem]">
      <div class="whitespace-pre-wrap">
        <i18n-t keypath="repository.sql-review-ci-restore-modal">
          <template #vcs>
            {{
              vcs.type === ExternalVersionControl_Type.GITLAB
                ? "GitLab CI"
                : "GitHub Action"
            }}
          </template>
          <template #repository>
            <a class="normal-link" :href="repository.webUrl" target="_blank">{{
              repository.fullPath
            }}</a>
          </template>
        </i18n-t>
      </div>

      <div class="flex justify-end pt-4 gap-x-2">
        <button
          type="button"
          class="btn-normal"
          @click.prevent="onSQLReviewCIModalClose"
        >
          {{ $t("common.close") }}
        </button>
      </div>
    </div>
  </BBModal>
  <BBModal
    v-if="supportSQLReviewCI && state.showDisableSQLReviewCIModal"
    class="relative overflow-hidden"
    :title="$t('repository.sql-review-ci-remove')"
    @close="state.showDisableSQLReviewCIModal = false"
  >
    <div class="space-y-4 max-w-[32rem]">
      <div class="whitespace-pre-wrap">
        <i18n-t keypath="repository.sql-review-ci-remove-modal">
          <template #vcs>
            {{
              vcs.type === ExternalVersionControl_Type.GITLAB
                ? "GitLab CI"
                : "GitHub Action"
            }}
          </template>
          <template #repository>
            <a class="normal-link" :href="repository.webUrl" target="_blank">{{
              repository.fullPath
            }}</a>
          </template>
        </i18n-t>
      </div>

      <div class="flex justify-end pt-4 gap-x-2">
        <button
          type="button"
          class="btn-normal"
          @click.prevent="state.showDisableSQLReviewCIModal = false"
        >
          {{ $t("common.close") }}
        </button>
      </div>
    </div>
  </BBModal>
  <FeatureModal
    feature="bb.feature.vcs-sql-review"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, watch } from "vue";
import isEmpty from "lodash-es/isEmpty";
import { ExternalRepositoryInfo, RepositoryConfig } from "../types";
import { useI18n } from "vue-i18n";
import {
  hasFeature,
  pushNotification,
  useProjectV1Store,
  useRepositoryV1Store,
} from "@/store";
import { Project, SchemaChange } from "@/types/proto/v1/project_service";
import { cloneDeep } from "lodash-es";
import {
  ProjectGitOpsInfo,
  ExternalVersionControl,
  ExternalVersionControl_Type,
} from "@/types/proto/v1/externalvs_service";

interface LocalState {
  repositoryConfig: RepositoryConfig;
  schemaChangeType: SchemaChange;
  showFeatureModal: boolean;
  showSetupSQLReviewCIModal: boolean;
  showSetupSQLReviewCIFailureModal: boolean;
  showLoadingSQLReviewPRModal: boolean;
  showDisableSQLReviewCIModal: boolean;
  showRestoreSQLReviewCIModal: boolean;
  sqlReviewCIPullRequestURL: string;
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
    type: Object as PropType<ExternalVersionControl>,
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
    branchFilter: props.repository.branchFilter,
    filePathTemplate: props.repository.filePathTemplate,
    schemaPathTemplate: props.repository.schemaPathTemplate,
    sheetPathTemplate: props.repository.sheetPathTemplate,
    enableSQLReviewCI: props.repository.enableSqlReviewCi,
  },
  schemaChangeType: props.project.schemaChange,
  showFeatureModal: false,
  showSetupSQLReviewCIModal: false,
  showSetupSQLReviewCIFailureModal: false,
  showLoadingSQLReviewPRModal: false,
  showDisableSQLReviewCIModal: false,
  showRestoreSQLReviewCIModal: false,
  sqlReviewCIPullRequestURL: "",
  processing: false,
});

watch(
  () => props.repository,
  (cur) => {
    state.repositoryConfig = {
      baseDirectory: cur.baseDirectory,
      branchFilter: cur.branchFilter,
      filePathTemplate: cur.filePathTemplate,
      schemaPathTemplate: cur.schemaPathTemplate,
      sheetPathTemplate: cur.sheetPathTemplate,
      enableSQLReviewCI: cur.enableSqlReviewCi,
    };
  }
);

const repositoryInfo = computed((): ExternalRepositoryInfo => {
  return {
    externalId: props.repository.externalId,
    name: props.repository.name,
    fullPath: props.repository.fullPath,
    webUrl: props.repository.webUrl,
  };
});

const isProjectSchemaChangeTypeDDL = computed(() => {
  return state.schemaChangeType === SchemaChange.DDL;
});

const isProjectSchemaChangeTypeSDL = computed(() => {
  return state.schemaChangeType === SchemaChange.SDL;
});

const supportSQLReviewCI = computed(() => {
  const { type } = props.vcs;
  return (
    type == ExternalVersionControl_Type.GITHUB ||
    type === ExternalVersionControl_Type.GITLAB
  );
});

const allowUpdate = computed(() => {
  return (
    !state.processing &&
    !isEmpty(state.repositoryConfig.branchFilter) &&
    !isEmpty(state.repositoryConfig.filePathTemplate) &&
    (props.repository.branchFilter !== state.repositoryConfig.branchFilter ||
      props.repository.baseDirectory !== state.repositoryConfig.baseDirectory ||
      props.repository.filePathTemplate !==
        state.repositoryConfig.filePathTemplate ||
      props.repository.schemaPathTemplate !==
        state.repositoryConfig.schemaPathTemplate ||
      props.repository.sheetPathTemplate !==
        state.repositoryConfig.sheetPathTemplate ||
      props.repository.enableSqlReviewCi !==
        state.repositoryConfig.enableSQLReviewCI ||
      props.project.schemaChange !== state.schemaChangeType)
  );
});

const disableSQLReviewCI = computed(() => {
  return (
    props.repository.enableSqlReviewCi &&
    !state.repositoryConfig.enableSQLReviewCI
  );
});

const onSQLReviewCIModalClose = () => {
  state.showRestoreSQLReviewCIModal = false;
  restoreToUIWorkflowType(false);
};

const restoreToUIWorkflowType = async (checkSQLReviewCI: boolean) => {
  if (checkSQLReviewCI && props.repository.enableSqlReviewCi) {
    state.showRestoreSQLReviewCIModal = true;
    return;
  }
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

const doUpdate = async () => {
  if (
    state.repositoryConfig.enableSQLReviewCI &&
    !props.repository.enableSqlReviewCi &&
    !hasFeature("bb.feature.vcs-sql-review")
  ) {
    state.showFeatureModal = true;
    return;
  }

  if (state.processing) {
    return;
  }
  state.processing = true;

  const needSetupCI =
    !props.repository.enableSqlReviewCi &&
    state.repositoryConfig.enableSQLReviewCI;

  const repositoryPatch: Partial<ProjectGitOpsInfo> = {};
  if (props.repository.branchFilter != state.repositoryConfig.branchFilter) {
    repositoryPatch.branchFilter = state.repositoryConfig.branchFilter;
  }
  if (props.repository.baseDirectory != state.repositoryConfig.baseDirectory) {
    repositoryPatch.baseDirectory = state.repositoryConfig.baseDirectory;
  }
  if (
    props.repository.filePathTemplate != state.repositoryConfig.filePathTemplate
  ) {
    repositoryPatch.filePathTemplate = state.repositoryConfig.filePathTemplate;
  }
  if (
    props.repository.schemaPathTemplate !=
    state.repositoryConfig.schemaPathTemplate
  ) {
    repositoryPatch.schemaPathTemplate =
      state.repositoryConfig.schemaPathTemplate;
  }
  if (
    props.repository.sheetPathTemplate !=
    state.repositoryConfig.sheetPathTemplate
  ) {
    repositoryPatch.sheetPathTemplate =
      state.repositoryConfig.sheetPathTemplate;
  }
  if (
    props.repository.enableSqlReviewCi !=
    state.repositoryConfig.enableSQLReviewCI
  ) {
    repositoryPatch.enableSqlReviewCi =
      state.repositoryConfig.enableSQLReviewCI;
  }

  try {
    const disableSQLReview = disableSQLReviewCI.value;

    await repositoryV1Store.upsertRepository(
      props.project.name,
      repositoryPatch
    );
    // Update project schemaChangeType field firstly.
    if (state.schemaChangeType !== props.project.schemaChange) {
      const projectPatch = cloneDeep(props.project);
      projectPatch.schemaChange = state.schemaChangeType;
      await projectV1Store.updateProject(projectPatch, ["schema_change"]);
    }

    if (needSetupCI) {
      createSQLReviewCI();
    } else if (disableSQLReview) {
      state.showDisableSQLReviewCIModal = true;
    }

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
