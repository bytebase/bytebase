<template>
  <!-- eslint-disable vue/no-mutating-props -->

  <div class="project-tenant-view">
    <template v-if="!!project">
      <template v-if="deployment?.id === UNKNOWN_ID">
        <i18n-t
          tag="p"
          keypath="deployment-config.project-has-no-deployment-config"
        >
          <template #go>
            <router-link
              :to="{
                path: `/project/${projectSlug(project)}`,
                hash: '#deployment-config',
              }"
              active-class=""
              exact-active-class=""
              class="px-1 underline hover:bg-link-hover"
              @click="$emit('dismiss')"
            >
              {{ $t("deployment-config.go-and-config") }}
            </router-link>
          </template>
        </i18n-t>
      </template>
      <template v-else>
        <div v-if="databaseList.length === 0" class="textinfolabel">
          <i18n-t keypath="project.overview.no-db-prompt" tag="p">
            <template #newDb>
              <span class="text-main">{{ $t("quick-action.new-db") }}</span>
            </template>
            <template #transferInDb>
              <span class="text-main">
                {{ $t("quick-action.transfer-in-db") }}
              </span>
            </template>
          </i18n-t>
        </div>
        <template v-else>
          <div class="flex justify-end items-center pb-2">
            <YAxisRadioGroup v-model:label="label" class="text-sm" />
          </div>

          <DeployDatabaseTable
            :database-list="databaseList"
            :label="label"
            :environment-list="environmentList"
            :deployment="deployment!"
          />
        </template>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import { computed, watchEffect, ref, h } from "vue";
import { Translation, useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";
import type {
  Database,
  DatabaseId,
  Environment,
  LabelKeyType,
  Project,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import { DeployDatabaseTable } from "../TenantDatabaseTable";
import { getPipelineFromDeploymentSchedule, projectSlug } from "@/utils";
import { useDeploymentStore } from "@/store";
import { useOverrideSubtitle } from "@/bbkit/BBModal.vue";

export type State = {
  selectedDatabaseIdListForTenantMode: Set<DatabaseId>;
  deployingTenantDatabaseList: DatabaseId[];
};

const props = defineProps<{
  databaseList: Database[];
  environmentList: Environment[];
  project?: Project;
  state: State;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const { t } = useI18n();

const deploymentStore = useDeploymentStore();

const fetchData = () => {
  if (props.project) {
    deploymentStore.fetchDeploymentConfigByProjectId(props.project.id);
  }
};

watchEffect(fetchData);

const label = ref<LabelKeyType>("bb.environment");

const deployment = computed(() => {
  if (props.project) {
    return deploymentStore.getDeploymentConfigByProjectId(props.project.id);
  } else {
    return undefined;
  }
});

watchEffect(() => {
  if (!deployment.value) return;
  const { databaseList } = props;

  // calculate the deployment matching to preview the pipeline
  const stages = getPipelineFromDeploymentSchedule(
    databaseList,
    deployment.value.schedule
  );

  // flatten all stages' database id list
  // these databases are to be deployed
  const databaseIdList = stages.flatMap((stage) => stage.map((db) => db.id));
  props.state.deployingTenantDatabaseList = databaseIdList;
});

useOverrideSubtitle(() => {
  return h(
    Translation,
    {
      tag: "p",
      class: "textinfolabel",
      keypath: "deployment-config.pipeline-generated-from-deployment-config",
    },
    {
      deployment_config: () =>
        h(
          RouterLink,
          {
            to: {
              path: `/project/${projectSlug(props.project!)}`,
              hash: "#databases",
            },
            activeClass: "",
            exactActiveClass: "",
            class: "underline hover:bg-link-hover",
            onClick: () => emit("dismiss"),
          },
          {
            default: () => t("common.deployment-config"),
          }
        ),
    }
  );
});
</script>

<style scoped lang="postcss">
.project-tenant-view :global(.n-collapse-item) {
  @apply mt-0 !important;
}

.project-tenant-view
  :global(.n-collapse-item.n-collapse-item--active + .n-collapse-item) {
  @apply border-0 !important;
}

.project-tenant-view :global(.n-collapse-item__header) {
  @apply pt-4 pb-4 border-control-light !important;
}

.project-tenant-view :global(.n-collapse-item__content-inner) {
  @apply pt-0 !important;
}
</style>
