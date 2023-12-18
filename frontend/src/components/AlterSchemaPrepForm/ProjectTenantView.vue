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
            :label="label"
            :environment-list="environmentList"
            :deployment="deploymentConfig"
          />
        </template>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { RouterLink } from "vue-router";
import { useDeploymentConfigV1ByProject } from "@/store";
import type { ComposedDatabase } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import { Project } from "@/types/proto/v1/project_service";
import { projectV1Slug } from "@/utils";
import { DeployDatabaseTable } from "../TenantDatabaseTable";

const props = defineProps<{
  label: string;
  databaseList: ComposedDatabase[];
  environmentList: Environment[];
  project?: Project;
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();

const { deploymentConfig, ready } = useDeploymentConfigV1ByProject(
  computed(() => {
    return props.project?.name ?? "projects/-1";
  })
);
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
