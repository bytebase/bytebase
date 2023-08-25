<template>
  <div class="flex flex-row space-x-2 items-center">
    <ProjectV1Name :project="project" :link="false" tag="div" />
    <div
      v-if="
        showTenantIcon && project.tenantMode === TenantMode.TENANT_MODE_ENABLED
      "
      class="tooltip-wrapper"
    >
      <span class="tooltip whitespace-nowrap">
        {{ $t("project.mode.batch") }}
      </span>
      <TenantIcon class="w-4 h-4 text-control" />
    </div>
    <div class="tooltip-wrapper">
      <svg
        v-if="project.workflow === Workflow.UI"
        class="w-4 h-4"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
      ></svg>
      <template v-else-if="project.workflow === Workflow.VCS">
        <span v-if="mode === 'ALL_SHORT'" class="tooltip w-40">
          {{ $t("alter-schema.vcs-info") }}
        </span>
        <span v-else class="tooltip whitespace-nowrap">
          {{ $t("database.gitops-enabled") }}
        </span>

        <GitIcon class="w-4 h-4 text-control hover:text-control-hover" />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  Project,
  TenantMode,
  Workflow,
} from "@/types/proto/v1/project_service";

defineProps<{
  project: Project;
  mode: string;
  showTenantIcon: boolean;
}>();
</script>
