<template>
  <div class="flex flex-row items-center flex-wrap gap-2">
    <NTag v-for="role in workspaceLevelRoles" :key="role">
      <template #avatar>
        <NTooltip>
          <template #trigger>
            <Building2Icon class="w-4 h-auto" />
          </template>
          {{ $t("project.members.workspace-level-roles") }}
        </NTooltip>
      </template>
      {{ displayRoleTitle(role) }}
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
import { displayRoleTitle, sortRoles, isBindingPolicyExpired } from "@/utils";
import type { ProjectRole } from "../../types";

const props = defineProps<{
  projectRole: ProjectRole;
}>();

const workspaceLevelRoles = computed(() => {
  return sortRoles(props.projectRole.workspaceLevelProjectRoles);
});

const projectRoleBindings = computed(() => {
  return orderBy(props.projectRole.projectRoleBindings, ["role"]);
});
</script>
