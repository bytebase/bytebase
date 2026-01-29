<template>
  <div class="flex flex-row items-center flex-wrap gap-2">
    <RoleCell
      v-for="r in workspaceLevelRoles"
      :key="r"
      :binding="create(BindingSchema, {
        role: r,
      })"
      :scope="'workspace'"
    />
    <RoleCell
      v-for="binding in projectRoleBindings"
      :key="binding.role"
      :binding="binding" :scope="'project'"
    />
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { orderBy } from "lodash-es";
import { computed } from "vue";
import { type Binding, BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import { isBindingPolicyExpired, sortRoles } from "@/utils";
import type { MemberRole } from "../../types";
import RoleCell from "./RoleCell.vue";

const props = defineProps<{
  role: MemberRole;
}>();

const workspaceLevelRoles = computed(() => {
  return sortRoles([...props.role.workspaceLevelRoles]);
});

const projectRoleBindings = computed(() => {
  const roleMap = new Map<string, { expired: boolean; binding: Binding }>();
  for (const binding of props.role.projectRoleBindings) {
    const isExpired = isBindingPolicyExpired(binding);
    if (
      !roleMap.has(binding.role) ||
      (roleMap.get(binding.role)?.expired && !isExpired)
    ) {
      roleMap.set(binding.role, {
        expired: isExpired,
        binding,
      });
    }
  }

  return orderBy(
    [...roleMap.values()].map((item) => item.binding),
    ["role"]
  );
});
</script>
