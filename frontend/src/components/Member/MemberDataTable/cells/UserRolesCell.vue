<template>
  <div class="flex flex-row items-center flex-wrap gap-2">
    <NTag v-for="r in workspaceLevelRoles" :key="r">
      <template #avatar>
        <NTooltip>
          <template #trigger>
            <Building2Icon class="w-4 h-auto" />
          </template>
          {{ $t("project.members.workspace-level-roles") }}
        </NTooltip>
      </template>
      {{ displayRoleTitle(r) }}
    </NTag>
    <NTag
      v-for="binding in projectRoleBindings"
      :key="binding.role"
      :class="isBindingPolicyExpired(binding) ? 'line-through' : ''"
    >
      {{ displayRoleTitle(binding.role) }}
    </NTag>
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { Building2Icon } from "lucide-vue-next";
import { NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { displayRoleTitle, isBindingPolicyExpired, sortRoles } from "@/utils";
import type { MemberRole } from "../../types";

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
