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
import { orderBy, uniqBy } from "lodash-es";
import { Building2Icon } from "lucide-vue-next";
import { NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import { displayRoleTitle, sortRoles, isBindingPolicyExpired } from "@/utils";
import type { MemberRole } from "../../types";

const props = defineProps<{
  role: MemberRole;
}>();

const workspaceLevelRoles = computed(() => {
  return sortRoles([...props.role.workspaceLevelRoles]);
});

const projectRoleBindings = computed(() => {
  return orderBy(
    uniqBy(props.role.projectRoleBindings, (binding) => binding.role),
    ["role"]
  );
});
</script>
