<template>
  <div class="max-w-[60rem] mx-auto">
    <div class="text-lg font-medium leading-7 text-main">
      {{ $t("project.db-name-template") }}
    </div>
    <div class="textinfolabel">
      <i18n-t keypath="label.db-name-template-tips">
        <template #placeholder>
          <!-- prettier-ignore -->
          <code v-pre class="text-xs font-mono bg-control-bg">{{DB_NAME}}</code>
        </template>
        <template #link>
          <a
            class="normal-link inline-flex items-center"
            href="https://bytebase.com/docs/tenant-database-management#database-name-template?source=console"
            target="__BLANK"
          >
            {{ $t("common.learn-more") }}
            <heroicons-outline:external-link class="w-4 h-4 ml-1" />
          </a>
        </template>
      </i18n-t>
    </div>
    <div class="mt-3 space-y-2">
      <div>
        <input
          v-model="state.dbNameTemplate"
          type="text"
          class="textfield w-full"
          :disabled="!state.isEditingDBNameTemplate"
        />
      </div>
      <div class="flex items-center justify-end gap-x-2">
        <button
          v-if="!state.isEditingDBNameTemplate"
          :disabled="!allowEdit"
          class="btn-normal"
          @click="beginEditDBNameTemplate"
        >
          {{ $t("common.edit") }}
        </button>
        <template v-if="state.isEditingDBNameTemplate">
          <button class="btn-normal" @click="cancelEditDBNameTemplate">
            {{ $t("common.cancel") }}
          </button>
          <button
            class="btn-primary"
            :disabled="!allowUpdateDBNameTemplate"
            @click="confirmEditDBNameTemplate"
          >
            {{ $t("common.update") }}
          </button>
        </template>
      </div>
    </div>

    <div class="text-lg font-medium leading-7 text-main mt-6 border-t pt-4">
      {{ $t("common.deployment-config") }}
    </div>

    <BBAttention
      v-if="state.deployment?.id === EMPTY_ID"
      :style="'WARN'"
      :title="$t('common.deployment-config')"
      :description="$t('deployment-config.this-is-example-deployment-config')"
    >
    </BBAttention>

    <div v-if="state.deployment">
      <DeploymentConfigTool
        :schedule="state.deployment.schedule"
        :allow-edit="allowEdit"
        :database-list="databaseList"
      />
      <div class="pt-4 border-t flex justify-between items-center">
        <div class="flex items-center space-x-2">
          <button v-if="allowEdit" class="btn-normal" @click="addStage">
            {{ $t("deployment-config.add-stage") }}
          </button>
        </div>
        <div class="flex items-center space-x-2">
          <button
            v-if="allowEdit"
            class="btn-normal"
            :disabled="!isDeploymentConfigDirty"
            @click="revertDeploymentConfig"
          >
            {{ $t("common.revert") }}
          </button>
          <NPopover v-if="allowEdit" :disabled="!state.error" trigger="hover">
            <template #trigger>
              <div
                class="btn-primary"
                :class="
                  !allowUpdateDeploymentConfig
                    ? 'bg-accent opacity-50 cursor-not-allowed'
                    : ''
                "
                @click="updateDeploymentConfig"
              >
                {{ $t("common.update") }}
              </div>
            </template>

            <span v-if="state.error" class="text-error">
              {{ $t(state.error) }}
            </span>
          </NPopover>
        </div>
      </div>

      <div class="mt-6">
        <div class="text-lg font-medium leading-7 text-main border-t pt-4">
          {{ $t("deployment-config.preview-deployment-pipeline") }}
        </div>
        <DeploymentMatrix
          class="w-full mt-4 !px-0 overflow-x-auto"
          :project="project"
          :deployment="state.deployment"
          :database-list="databaseList"
          :environment-list="environmentList"
        />
      </div>
    </div>

    <div v-else class="flex justify-center items-center py-10">
      <BBSpin />
    </div>
  </div>
</template>

<script lang="ts">
import {
  computed,
  defineComponent,
  nextTick,
  PropType,
  reactive,
  watch,
  watchEffect,
} from "vue";
import { cloneDeep, isEqual } from "lodash-es";
import { useI18n } from "vue-i18n";
import { NPopover, useDialog } from "naive-ui";
import {
  Project,
  DeploymentConfig,
  EMPTY_ID,
  DeploymentConfigPatch,
  LabelSelectorRequirement,
} from "../types";
import DeploymentConfigTool, { DeploymentMatrix } from "./DeploymentConfigTool";
import { validateDeploymentConfig } from "../utils";
import {
  pushNotification,
  useDatabaseStore,
  useDeploymentStore,
  useEnvironmentList,
  useEnvironmentStore,
  useProjectStore,
} from "@/store";

type LocalState = {
  deployment: DeploymentConfig | undefined;
  originalDeployment: DeploymentConfig | undefined;
  error: string | undefined;
  isEditingDBNameTemplate: boolean;
  dbNameTemplate: string;
};

export default defineComponent({
  name: "ProjectDeploymentConfigurationPanel",
  components: { DeploymentConfigTool, DeploymentMatrix, NPopover },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
    allowEdit: {
      default: true,
      type: Boolean,
    },
  },
  setup(props) {
    const databaseStore = useDatabaseStore();
    const deploymentStore = useDeploymentStore();
    const { t } = useI18n();
    const dialog = useDialog();

    const state = reactive<LocalState>({
      deployment: undefined,
      originalDeployment: undefined,
      error: undefined,
      isEditingDBNameTemplate: false,
      dbNameTemplate: props.project.dbNameTemplate,
    });

    const isDeploymentConfigDirty = computed((): boolean => {
      return !isEqual(state.deployment, state.originalDeployment);
    });

    const allowUpdateDeploymentConfig = computed((): boolean => {
      if (state.error) return false;
      if (!isDeploymentConfigDirty.value) return false;
      return true;
    });

    const prepareList = () => {
      useEnvironmentStore().fetchEnvironmentList();
      databaseStore.fetchDatabaseListByProjectId(props.project.id);
    };

    const environmentList = useEnvironmentList();

    const databaseList = computed(() =>
      databaseStore.getDatabaseListByProjectId(props.project.id)
    );

    watchEffect(prepareList);

    const resetStates = async () => {
      await nextTick(); // Waiting for all watchers done
      state.error = undefined;
    };

    watchEffect(async () => {
      const dep = await deploymentStore.fetchDeploymentConfigByProjectId(
        props.project.id
      );
      // We clone the saved deployment-config
      // <DeploymentConfigTool /> will mutate `state.deployment` directly
      // when update button clicked, we save the draft to backend.
      state.deployment = cloneDeep(dep);
      // clone the object to the backup
      state.originalDeployment = cloneDeep(state.deployment);
      // clean up error and dirty status
      resetStates();
    });

    const addStage = () => {
      if (!state.deployment) return;
      const rule: LabelSelectorRequirement = {
        key: "bb.environment",
        operator: "In",
        values: [],
      };
      if (environmentList.value.length > 0) {
        rule.values.push(environmentList.value[0].name);
      }

      state.deployment.schedule.deployments.push({
        name: "New Stage",
        spec: {
          selector: {
            matchExpressions: [rule],
          },
        },
      });
    };

    const validate = () => {
      if (!state.deployment) return;
      state.error = validateDeploymentConfig(state.deployment);
    };

    const revertDeploymentConfig = () => {
      dialog.create({
        positiveText: t("common.confirm"),
        negativeText: t("common.cancel"),
        title: t("deployment-config.confirm-to-revert"),
        closable: false,
        maskClosable: false,
        closeOnEsc: false,
        onNegativeClick: () => {
          // nothing to do
        },
        onPositiveClick: () => {
          state.deployment = cloneDeep(state.originalDeployment);
          resetStates();
        },
      });
    };

    const updateDeploymentConfig = async () => {
      if (!state.deployment) return;
      if (!allowUpdateDeploymentConfig.value) return;

      const deploymentConfigPatch: DeploymentConfigPatch = {
        payload: JSON.stringify(state.deployment.schedule),
      };
      await deploymentStore.patchDeploymentConfigByProjectId({
        projectId: props.project.id,
        deploymentConfigPatch,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("deployment-config.update-success"),
      });

      // clone the updated version to the backup
      state.originalDeployment = cloneDeep(state.deployment);
      // clean up error status
      resetStates();
    };

    watch(
      () => state.deployment,
      () => {
        validate();
      },
      { deep: true }
    );

    const dbNameTemplateTips = computed(() =>
      t("label.db-name-template-tips", {
        placeholder: "{{DB_NAME}}",
      })
    );

    const allowUpdateDBNameTemplate = computed(() => {
      return state.dbNameTemplate !== props.project.dbNameTemplate;
    });

    const beginEditDBNameTemplate = () => {
      state.dbNameTemplate = props.project.dbNameTemplate;
      state.isEditingDBNameTemplate = true;
    };

    const cancelEditDBNameTemplate = () => {
      state.dbNameTemplate = props.project.dbNameTemplate;
      state.isEditingDBNameTemplate = false;
    };

    const confirmEditDBNameTemplate = async () => {
      try {
        await useProjectStore().patchProject({
          projectId: props.project.id,
          projectPatch: {
            dbNameTemplate: state.dbNameTemplate,
          },
        });
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("project.successfully-updated-db-name-template"),
        });
      } catch {
        state.dbNameTemplate = props.project.dbNameTemplate;
      } finally {
        state.isEditingDBNameTemplate = false;
      }
    };

    return {
      EMPTY_ID,
      state,
      isDeploymentConfigDirty,
      allowUpdateDeploymentConfig,
      environmentList,
      databaseList,
      addStage,
      revertDeploymentConfig,
      updateDeploymentConfig,
      dbNameTemplateTips,
      allowUpdateDBNameTemplate,
      beginEditDBNameTemplate,
      cancelEditDBNameTemplate,
      confirmEditDBNameTemplate,
    };
  },
});
</script>
