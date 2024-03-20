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
    <NTag v-for="binding in projectRoleBindings" :key="binding.role">
      {{ displayRoleTitle(binding.role) }}
    </NTag>
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { Building2Icon } from "lucide-vue-next";
import { NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import { displayRoleTitle, sortRoles } from "@/utils";
import { ProjectMember } from "../../types";

const props = defineProps<{
  projectMember: ProjectMember;
}>();

const workspaceLevelRoles = computed(() => {
  return sortRoles(props.projectMember.workspaceLevelProjectRoles);
});

const projectRoleBindings = computed(() => {
  return orderBy(props.projectMember.projectRoleBindings, ["role"]);
});
</script>
