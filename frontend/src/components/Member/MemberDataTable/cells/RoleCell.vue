<template>
  <component
    :is="hasRolePermission ? 'router-link' : 'div'"
    :to="{
      name: WORKSPACE_ROUTE_ROLES,
      query: {
        role: binding.role,
      },
    }"
  >
    <NTag
      :bordered="bordered"
      :class="isBindingPolicyExpired(binding) ? 'line-through' : ''"
      style="background-color: transparent !important;"
    >
      <div class="flex items-center gap-x-1 cursor-pointer">
        <NTooltip v-if="scope === 'workspace'">
          <template #trigger>
            <Building2Icon class="w-4 h-auto mr-1" />
          </template>
          {{ $t("project.members.workspace-level-roles") }}
        </NTooltip>
        {{ displayRoleTitle(binding.role) }}
        <span v-if="count !== undefined" class="font-normal text-control-light">
          ({{ count }})
        </span>
      </div>
    </NTag>
  </component>
</template>

<script lang="tsx" setup>
import { Building2Icon } from "lucide-vue-next";
import { NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import { WORKSPACE_ROUTE_ROLES } from "@/router/dashboard/workspaceRoutes";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import {
  displayRoleTitle,
  hasWorkspacePermissionV2,
  isBindingPolicyExpired,
} from "@/utils";

defineProps<{
  binding: Binding;
  scope: "workspace" | "project";
  count?: number;
  bordered: boolean;
}>();

const hasRolePermission = computed(() =>
  hasWorkspacePermissionV2("bb.roles.list")
);
</script>