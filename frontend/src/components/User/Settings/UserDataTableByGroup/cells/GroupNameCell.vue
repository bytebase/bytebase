<template>
  <div class="flex items-center space-x-2">
    <UsersIcon v-if="showIcon" class="w-8" />
    <div>
      <div class="flex items-center">
        <router-link
          v-if="allowGetGroup && link"
          :to="{
            name: WORKSPACE_ROUTE_MEMBERS,
            query: {
              name: group.name,
            },
          }"
          class="normal-link font-medium"
        >
          {{ group.title }}
        </router-link>
        <span v-else class="font-medium">{{ group.title }}</span>
        <span class="ml-1 font-normal text-control-light">
          {{
            `(${$t("settings.members.groups.n-members", { n: group.members.length })})`
          }}
        </span>
        <UserRolesCell v-if="role" class="ml-3" :project-role="role" />
      </div>
      <span v-if="showEmail" class="textinfolabel text-sm">
        {{ extractGroupEmail(group.name) }}
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { UsersIcon } from "lucide-vue-next";
import { computed } from "vue";
import UserRolesCell from "@/components/ProjectMember/ProjectMemberDataTable/cells/UserRolesCell.vue";
import type { ProjectRole } from "@/components/ProjectMember/types";
import { WORKSPACE_ROUTE_MEMBERS } from "@/router/dashboard/workspaceRoutes";
import { extractGroupEmail, extractUserEmail, useCurrentUserV1 } from "@/store";
import { Group } from "@/types/proto/v1/group";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = withDefaults(
  defineProps<{
    group: Group;
    role?: ProjectRole;
    showIcon?: boolean;
    showEmail?: boolean;
    link?: boolean;
  }>(),
  {
    showIcon: true,
    showEmail: true,
    role: undefined,
    link: true,
  }
);

const currentUser = useCurrentUserV1();

const allowGetGroup = computed(() => {
  if (
    props.group.members.find(
      (member) => extractUserEmail(member.member) === currentUser.value.name
    )
  ) {
    return true;
  }
  return hasWorkspacePermissionV2(currentUser.value, "bb.groups.get");
});
</script>
