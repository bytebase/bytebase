<template>
  <StepTab
    :sticky="true"
    :current-index="state.currentStep"
    :step-list="STEP_LIST"
    :allow-next="allowNext"
    :cancel-title="$t('setup.skip-setup')"
    :finish-title="$t('common.confirm')"
    @cancel="() => onCancel('/')"
    @update:current-index="changeStepIndex"
    @finish="tryFinishSetup"
  >
    <template #0>
      <div class="w-full flex flex-col gap-6 py-4">
        <div class="flex flex-col gap-y-2">
          <p>{{ $t("setup.purposes.self") }}</p>
          <NRadioGroup v-model:value="state.purpose">
            <NSpace vertical>
              <NRadio :value="'edit-schema'">
                {{ $t("setup.purposes.alter-schema") }}
              </NRadio>
              <NRadio :value="'query-data'">
                {{ $t("setup.purposes.query-data") }}
              </NRadio>
            </NSpace>
          </NRadioGroup>
        </div>
        <div class="flex flex-col gap-y-2">
          <p>{{ $t("setup.workflow.self") }}</p>
          <NRadioGroup v-model:value="state.workflow">
            <NSpace vertical>
              <NRadio :value="'team'">
                {{ $t("setup.workflow.team") }}
              </NRadio>
              <NRadio :value="'simple'">
                {{ $t("setup.workflow.simple") }}
              </NRadio>
            </NSpace>
          </NRadioGroup>
        </div>
      </div>
    </template>
    <template #1>
      <div class="w-full flex flex-col gap-2 py-4">
        <NRadioGroup v-model:value="state.data">
          <NSpace vertical size="large">
            <NRadio :value="'self-setup'">
              <div>
                <p class="font-medium">{{ $t("setup.data.self-setup") }}</p>
                <div>
                  <BBTextField
                    v-model:value="state.project.title"
                    class="mt-2 mb-1 w-full"
                    :required="true"
                    :placeholder="$t('project.create-modal.project-name')"
                  />
                  <ResourceIdField
                    ref="resourceIdField"
                    editing-class="mt-2"
                    resource-type="project"
                    :value="state.resourceId"
                    :resource-title="state.project.title"
                    :fetch-resource="
                      (id) =>
                        projectV1Store.getOrFetchProjectByName(
                          `${projectNamePrefix}${id}`,
                          true /* silent */
                        )
                    "
                    @update:value="state.resourceId = $event"
                  />
                </div>
              </div>
            </NRadio>
            <NRadio :value="'builtin-sample'">
              <div>
                <p class="font-medium">
                  {{ $t("setup.data.built-in") }}
                </p>
                <div>
                  {{ $t("setup.data.built-in-desc") }}
                </div>
              </div>
            </NRadio>
          </NSpace>
        </NRadioGroup>
      </div>
    </template>
    <template #2>
      <div class="py-4">
        <WorkspaceMode v-model:mode="state.mode" />
      </div>
    </template>
  </StepTab>
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { NRadio, NRadioGroup, NSpace } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { type RouteLocationRaw, useRouter } from "vue-router";
import { BBTextField } from "@/bbkit";
import { StepTab } from "@/components/v2";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import {
  useActuatorV1Store,
  useProjectV1Store,
  useSettingV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { emptyProject } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  DatabaseChangeMode,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";
import WorkspaceMode from "./WorkspaceMode.vue";

interface LocalState {
  currentStep: number;
  purpose: "edit-schema" | "query-data";
  data: "self-setup" | "builtin-sample";
  workflow: "simple" | "team";
  mode: DatabaseChangeMode;
  project: Project;
  resourceId: string;
  loading: boolean;
}

const { t } = useI18n();
const router = useRouter();
const projectV1Store = useProjectV1Store();
const actuatorV1Store = useActuatorV1Store();
const settingStore = useSettingV1Store();

const state = reactive<LocalState>({
  currentStep: 0,
  purpose: "edit-schema",
  workflow: "team",
  data: "self-setup",
  mode: DatabaseChangeMode.PIPELINE,
  project: {
    ...emptyProject(),
    title: "New Project",
  },
  resourceId: "",
  loading: false,
});
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

const STEP_LIST = [
  { title: t("setup.basic-info") },
  { title: t("setup.self") },
  { title: t("settings.general.workspace.default-landing-page.self") },
];

const allowNext = computed(() => {
  if (state.currentStep === 1) {
    if (state.data === "self-setup") {
      return (
        !!state.project.title && (resourceIdField.value?.isValidated ?? false)
      );
    }
  } else if (state.currentStep === STEP_LIST.length - 1) {
    return !state.loading;
  }
  return true;
});

const changeStepIndex = (nextIndex: number) => {
  if (nextIndex === STEP_LIST.length - 1) {
    if (state.purpose === "query-data" && state.workflow === "simple") {
      state.mode = DatabaseChangeMode.EDITOR;
    } else {
      state.mode = DatabaseChangeMode.PIPELINE;
    }
  }

  state.currentStep = nextIndex;
};

const onCancel = (to: RouteLocationRaw) => {
  actuatorV1Store.onboardingState.isOnboarding = false;
  router.push(to);
};

const getHomePageByMode = (mode?: DatabaseChangeMode) => {
  if (mode === DatabaseChangeMode.EDITOR) {
    return {
      name: SQL_EDITOR_HOME_MODULE,
    };
  }
  return "/";
};

const tryFinishSetup = async () => {
  if (state.loading) {
    return;
  }
  state.loading = true;

  try {
    if (state.data === "self-setup") {
      await projectV1Store.createProject(state.project, state.resourceId);
    } else {
      await actuatorV1Store.setupSample();
    }
    await settingStore.updateWorkspaceProfile({
      payload: {
        databaseChangeMode: state.mode,
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.database_change_mode"],
      }),
    });
    onCancel(getHomePageByMode(state.mode));
  } finally {
    state.loading = false;
  }
};

onMounted(async () => {
  if (!actuatorV1Store.onboardingState.isOnboarding) {
    const profileSetting = await settingStore.fetchSettingByName(
      Setting_SettingName.WORKSPACE_PROFILE
    );
    return onCancel(
      getHomePageByMode(
        profileSetting?.value?.value?.case === "workspaceProfile"
          ? profileSetting.value.value.value.databaseChangeMode
          : undefined
      )
    );
  }
});
</script>
