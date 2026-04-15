<template>
  <div class="flex flex-row items-center flex-wrap gap-1">
    <RoleCell
      v-for="r in workspaceLevelRoles"
      :key="r"
      :binding="create(BindingSchema, {
        role: r,
      })"
      :bordered="true"
      :scope="'workspace'"
    />
    <RoleCell
      v-for="binding in projectRoleBindings"
      :key="binding.role"
      :bordered="true"
      :binding="binding" :scope="'project'"
    />
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { computed } from "vue";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import { sortRoles } from "@/utils";
import { getUniqueProjectRoleBindings } from "../../projectRoleBindings";
import type { MemberRole } from "../../types";
import RoleCell from "./RoleCell.vue";

const props = defineProps<{
  role: MemberRole;
}>();

const workspaceLevelRoles = computed(() => {
  return sortRoles([...props.role.workspaceLevelRoles]);
});

const projectRoleBindings = computed(() => {
  return getUniqueProjectRoleBindings(props.role.projectRoleBindings);
});
</script>
