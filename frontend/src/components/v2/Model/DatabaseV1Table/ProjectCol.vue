<template>
  <div class="flex flex-row space-x-2 items-center">
    <ProjectV1Name :project="project" :link="false" tag="div" />

    <NTooltip
      v-if="
        showTenantIcon && project.tenantMode === TenantMode.TENANT_MODE_ENABLED
      "
    >
      <template #trigger>
        <TenantIcon class="ml-1 text-control" />
      </template>
      <span class="whitespace-nowrap">
        {{ $t("project.mode.batch") }}
      </span>
    </NTooltip>

    <NTooltip v-if="project.workflow === Workflow.VCS">
      <template #trigger>
        <GitIcon class="ml-1 w-4 h-4 text-control" />
      </template>
      <span v-if="mode === 'ALL_SHORT'" class="w-40">
        {{ $t("alter-schema.vcs-info") }}
      </span>
      <span v-else class="tooltip whitespace-nowrap">
        {{ $t("database.gitops-enabled") }}
      </span>
    </NTooltip>

    <NTooltip v-if="project.state === State.DELETED">
      <template #trigger>
        <heroicons-outline:archive class="ml-1 w-4 h-4 text-control" />
      </template>
      <span class="whitespace-nowrap">
        {{ $t("common.archived") }}
      </span>
    </NTooltip>
  </div>
</template>

<script setup lang="ts">
import { State } from "@/types/proto/v1/common";
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
