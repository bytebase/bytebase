<template>
  <div class="max-w-3xl mx-auto">
    <template v-if="project.dbNameTemplate">
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
              href="https://docs.bytebase.com/features/tenant-database-management#database-name-template"
              target="__BLANK"
            >
              {{ $t("common.learn-more") }}
              <heroicons-outline:external-link class="w-4 h-4 ml-1" />
            </a>
          </template>
        </i18n-t>
      </div>
      <div class="mt-3">
        <input
          disabled
          type="text"
          class="textfield w-full"
          :value="project.dbNameTemplate"
        />
      </div>

      <div class="text-lg font-medium leading-7 text-main mt-6 border-t pt-4">
        {{ $t("common.deployment-config") }}
      </div>
    </template>

    <BBAttention
      v-if="deployment?.id === EMPTY_ID"
      :style="'WARN'"
      :title="$t('common.deployment-config')"
      :description="$t('deployment-config.this-is-example-deployment-config')"
    >
    </BBAttention>

    <div class="divide-y">
      <DeploymentConfigTool
        v-if="deployment"
        :schedule="deployment.schedule"
        :allow-edit="allowEdit"
        :label-list="availableLabelList"
        :database-list="databaseList"
      />
      <div class="pt-4 flex justify-between items-center">
        <div class="flex items-center space-x-2">
          <button v-if="allowEdit" class="btn-normal" @click="addStage">
            {{ $t("deployment-config.add-stage") }}
          </button>

          <button
            class="btn-normal"
            :disabled="!!state.error"
            @click="state.showPreview = true"
          >
            {{ $t("common.preview") }}
          </button>
        </div>
        <div class="flex items-center space-x-2">
          <NPopover v-if="allowEdit" :disabled="!state.error" trigger="hover">
            <template #trigger>
              <div
                class="btn-primary"
                :class="
                  state.error ? 'bg-accent opacity-50 cursor-not-allowed' : ''
                "
                @click="update"
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
    </div>

    <BBModal
      v-if="deployment && state.showPreview"
      :title="$t('deployment-config.preview-deployment-config')"
      @close="state.showPreview = false"
    >
      <DeploymentRoadmap
        :project="project"
        :deployment="deployment"
        :database-list="databaseList"
        :environment-list="environmentList"
        :label-list="labelList"
      />
    </BBModal>
  </div>
</template>

<script lang="ts">
import {
  computed,
  defineComponent,
  nextTick,
  PropType,
  reactive,
  ref,
  watch,
  watchEffect,
} from "vue";
import { useStore } from "vuex";
import {
  Project,
  Environment,
  Label,
  Database,
  AvailableLabel,
  DeploymentConfig,
  UNKNOWN_ID,
  EMPTY_ID,
  empty,
  DeploymentConfigPatch,
  LabelSelectorRequirement,
} from "../types";
import DeploymentConfigTool, {
  DeploymentRoadmap,
} from "./DeploymentConfigTool";
import { cloneDeep } from "lodash-es";
import { useI18n } from "vue-i18n";
import { NPopover } from "naive-ui";
import { generateDefaultSchedule, validateDeploymentConfig } from "../utils";

type LocalState = {
  error: string | undefined;
  ready: boolean;
  showPreview: boolean;
};

export default defineComponent({
  name: "ProjectDeploymentConfigurationPanel",
  components: { DeploymentConfigTool, DeploymentRoadmap, NPopover },
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
    const store = useStore();
    const { t } = useI18n();
    const deployment = ref<DeploymentConfig>();

    const state = reactive<LocalState>({
      ready: false,
      error: undefined,
      showPreview: false,
    });

    const prepareList = () => {
      store.dispatch("environment/fetchEnvironmentList");
      store.dispatch("label/fetchLabelList");
      store.dispatch("database/fetchDatabaseListByProjectId", props.project.id);
      store.dispatch(
        "deployment/fetchDeploymentConfigByProjectId",
        props.project.id
      );
    };

    const environmentList = computed(
      () => store.getters["environment/environmentList"]() as Environment[]
    );

    const labelList = computed(
      () => store.getters["label/labelList"]() as Label[]
    );

    const databaseList = computed(
      () =>
        store.getters["database/databaseListByProjectId"](
          props.project.id
        ) as Database[]
    );

    watchEffect(prepareList);

    const availableLabelList = computed((): AvailableLabel[] => {
      return labelList.value.map((label) => {
        return { key: label.key, valueList: [...label.valueList] };
      });
    });

    watchEffect(() => {
      const dep = store.getters["deployment/deploymentConfigByProjectId"](
        props.project.id
      ) as DeploymentConfig;
      if (dep.id === UNKNOWN_ID) {
        // if the project has no related deployment-config
        // just generate a "staged-by-env" example to users
        // this is not saved immediately, it's a draft
        // users need to edit and save it before creating a deployment issue
        if (environmentList.value.length > 0) {
          deployment.value = empty("DEPLOYMENT_CONFIG") as DeploymentConfig;
          deployment.value.schedule = generateDefaultSchedule(
            environmentList.value
          );
        }
      } else {
        // otherwise we clone the saved deployment-config
        // <DeploymentConfigTool /> will mutate `deployment.value` directly
        // when update button clicked, we save the draft to backend
        // we don't show a "cancel" button because if users don't want to save
        //   the draft, they can just leave the page without any saving action
        // even more we may deliver a confirm modal when leaving the page with a
        //   dirty but not saved draft
        deployment.value = cloneDeep(dep);
      }
      nextTick(() => {
        // then we reset the local state
        state.ready = true;
        state.error = undefined;
      });
    });

    const addStage = () => {
      if (!deployment.value) return;
      const rule: LabelSelectorRequirement = {
        key: "bb.environment",
        operator: "In",
        values: [],
      };
      if (environmentList.value.length > 0) {
        rule.values.push(environmentList.value[0].name);
      }

      deployment.value.schedule.deployments.push({
        name: "New Stage",
        spec: {
          selector: {
            matchExpressions: [rule],
          },
        },
      });
    };

    const validate = () => {
      if (!deployment.value) return;
      state.error = validateDeploymentConfig(deployment.value);
    };

    const update = () => {
      if (!deployment.value) return;
      if (state.error) return;

      const deploymentConfigPatch: DeploymentConfigPatch = {
        payload: JSON.stringify(deployment.value.schedule),
      };
      store.dispatch("deployment/patchDeploymentConfigByProjectId", {
        projectId: props.project.id,
        deploymentConfigPatch,
      });
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "SUCCESS",
        title: t("deployment-config.update-success"),
      });
    };

    watch(
      deployment,
      (dep) => {
        if (!dep) return;
        if (!state.ready) return;
        validate();
      },
      { deep: true }
    );

    const dbNameTemplateTips = computed(() =>
      t("label.db-name-template-tips", {
        placeholder: "{{DB_NAME}}",
      })
    );

    return {
      EMPTY_ID,
      state,
      environmentList,
      labelList,
      databaseList,
      availableLabelList,
      deployment,
      addStage,
      update,
      dbNameTemplateTips,
    };
  },
});
</script>
