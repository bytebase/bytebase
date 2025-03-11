<template>
  <div class="flex items-center space-x-2">
    <UsersIcon v-if="showIcon" class="w-8" />
    <div>
      <div class="flex items-center">
        <router-link
          v-if="allowGetGroup && link"
          :to="{
            name: WORKSPACE_ROUTE_USERS,
            query: {
              name: group.name,
            },
          }"
          class="normal-link font-medium"
        >
          {{ group.title }}
        </router-link>
        <span v-else class="font-medium">{{ group.title }}</span>
        <NTag
          v-if="group.source"
          size="small"
          round
          type="primary"
          class="ml-2"
        >
          {{ group.source }}
        </NTag>
        <span class="ml-2 font-normal text-control-light">
          {{
            `(${$t("settings.members.groups.n-members", { n: group.members.length })})`
          }}
        </span>
        <UserRolesCell v-if="role" class="ml-3" :role="role" />
      </div>
      <span v-if="showEmail" class="textinfolabel text-sm">
        {{ extractGroupEmail(group.name) }}
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { UsersIcon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed } from "vue";
import UserRolesCell from "@/components/Member/MemberDataTable/cells/UserRolesCell.vue";
import type { MemberRole } from "@/components/Member/types";
import { WORKSPACE_ROUTE_USERS } from "@/router/dashboard/workspaceRoutes";
import { extractGroupEmail, extractUserId, useCurrentUserV1 } from "@/store";
import { Group } from "@/types/proto/v1/group_service";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = withDefaults(
  defineProps<{
    group: Group;
    role?: MemberRole;
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
      (member) => extractUserId(member.member) === currentUser.value.name
    )
  ) {
    return true;
  }
  return hasWorkspacePermissionV2("bb.groups.get");
});
</script>
