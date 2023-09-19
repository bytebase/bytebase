<template>
  <!-- eslint-disable vue/no-mutating-props -->

  <div class="project-tenant-view">
    <template v-if="project && ready">
      <template v-if="deploymentConfig === undefined">
        <i18n-t
          tag="p"
          keypath="deployment-config.project-has-no-deployment-config"
        >
          <template #go>
            <router-link
              :to="{
                path: `/project/${projectV1Slug(project)}`,
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
          <DeployDatabaseTable
            :database-list="databaseList"
            :label="state.label"
            :environment-list="environmentList"
            :deployment="deploymentConfig"
          />
        </template>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import { computed, watchEffect } from "vue";
import { RouterLink } from "vue-router";
import { useDeploymentConfigV1ByProject } from "@/store";
import type { ComposedDatabase, LabelKeyType } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import { Project } from "@/types/proto/v1/project_service";
import { getPipelineFromDeploymentScheduleV1, projectV1Slug } from "@/utils";
import { DeployDatabaseTable } from "../TenantDatabaseTable";

export type ProjectTenantViewState = {
  selectedDatabaseIdListForTenantMode: Set<string>;
  deployingTenantDatabaseList: string[];
  label: LabelKeyType;
};

const props = defineProps<{
  databaseList: ComposedDatabase[];
  environmentList: Environment[];
  project?: Project;
  state: ProjectTenantViewState;
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();

const { deploymentConfig, ready } = useDeploymentConfigV1ByProject(
  computed(() => {
    return props.project?.name ?? "projects/-1";
  })
);

watchEffect(() => {
  if (!deploymentConfig.value) return;
  const { databaseList } = props;

  // calculate the deployment matching to preview the pipeline
  const stages = getPipelineFromDeploymentScheduleV1(
    databaseList,
    deploymentConfig.value.schedule
  );

  // flatten all stages' database id list
  // these databases are to be deployed
  const databaseIdList = stages.flatMap((stage) => stage.map((db) => db.uid));
  props.state.deployingTenantDatabaseList = databaseIdList;
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
