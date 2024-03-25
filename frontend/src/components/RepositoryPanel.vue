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
    <template v-if="isProjectSchemaChangeTypeDDL">
      <i18n-t keypath="repository.gitops-description-file-path">
        <template #fullPath>
          <a class="normal-link" :href="repository.webUrl" target="_blank">
            {{ repositoryFormattedFullPath }}
          </a>
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
            {{ repositoryFormattedFullPath }}
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
    @change-schema-change-type="
      (type: SchemaChange) => (state.schemaChangeType = type)
    "
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
import { cloneDeep } from "lodash-es";
import isEmpty from "lodash-es/isEmpty";
import { computed, PropType, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useProjectV1Store,
  useRepositoryV1Store,
} from "@/store";
import { getVCSUid } from "@/store/modules/v1/common";
import { Project, SchemaChange } from "@/types/proto/v1/project_service";
import {
  ProjectGitOpsInfo,
  VCSProvider,
  VCSProvider_Type,
} from "@/types/proto/v1/vcs_provider_service";
import { ExternalRepositoryInfo, RepositoryConfig } from "../types";

interface LocalState {
  repositoryConfig: RepositoryConfig;
  schemaChangeType: SchemaChange;
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
    branchFilter: props.repository.branchFilter,
    filePathTemplate: props.repository.filePathTemplate,
    schemaPathTemplate: props.repository.schemaPathTemplate,
  },
  schemaChangeType: props.project.schemaChange,
  showFeatureModal: false,
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

const isProjectSchemaChangeTypeDDL = computed(() => {
  return state.schemaChangeType === SchemaChange.DDL;
});

const isProjectSchemaChangeTypeSDL = computed(() => {
  return state.schemaChangeType === SchemaChange.SDL;
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
      props.project.schemaChange !== state.schemaChangeType)
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

  repositoryPatch.vcsUid = `${getVCSUid(props.vcs.name)}`;

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

  try {
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
