<template>
  <NTooltip :disabled="missedPermissions.length === 0">
    <template #trigger>
      <slot :disabled="missedPermissions.length > 0" />
    </template>
    <div class="flex flex-col gap-1">
      {{ project ? $t("common.missing-required-permission-for-resource", { resource: project.name }) : $t("common.missing-required-permission") }}
      <ul class="list-disc pl-4">
        <li v-for="permission in missedPermissions" :key="permission">
          {{ permission }}
        </li>
      </ul>
    </div>
  </NTooltip>
</template>

<script lang="tsx" setup>
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import type { Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  project?: Project;
  permissions: Permission[];
}>();

const missedPermissions = computed(() => {
  if (props.project) {
    return props.permissions.filter(
      (p) => !hasProjectPermissionV2(props.project!, p)
    );
  }
  return props.permissions.filter((p) => !hasWorkspacePermissionV2(p));
});
</script>