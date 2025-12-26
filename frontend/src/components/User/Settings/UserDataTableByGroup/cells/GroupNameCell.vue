<template>
  <div class="flex items-center gap-x-2">
    <UsersIcon v-if="showIcon" class="w-8" />
    <div>
      <div class="flex items-center gap-x-2">
        <NEllipsis
          :line-clamp="1"
          :tooltip="true"
          :class="deleted ? 'line-through' : ''"
        >
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
            <HighlightLabelText
              :text="group.title"
              :keyword="keyword"
            />
          </router-link>
          <HighlightLabelText
            v-else
            class="font-medium"
            :text="group.title"
            :keyword="keyword"
          />
        </NEllipsis>
        <NTag v-if="deleted" size="small" round type="error">
          {{ $t("common.deleted") }}
        </NTag>
        <NTag v-else-if="group.source" size="small" round type="primary">
          {{ group.source }}
        </NTag>
        <span v-if="showMember" class="font-normal text-control-light">
          {{
            `(${$t("settings.members.groups.n-members", { n: group.members.length })})`
          }}
        </span>
        <UserRolesCell v-if="role" :role="role" />
      </div>
      <NEllipsis
        v-if="showEmail && group.email"
        :line-clamp="1"
        :tooltip="true"
      >
        <HighlightLabelText
          class="textinfolabel text-sm"
          :text="group.email"
          :keyword="keyword"
        />
      </NEllipsis>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { UsersIcon } from "lucide-vue-next";
import { NEllipsis, NTag } from "naive-ui";
import { computed } from "vue";
import UserRolesCell from "@/components/Member/MemberDataTable/cells/UserRolesCell.vue";
import type { MemberRole } from "@/components/Member/types";
import { HighlightLabelText } from "@/components/v2";
import { WORKSPACE_ROUTE_USERS } from "@/router/dashboard/workspaceRoutes";
import { extractUserId, useCurrentUserV1 } from "@/store";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = withDefaults(
  defineProps<{
    group: Group;
    role?: MemberRole;
    showIcon?: boolean;
    showEmail?: boolean;
    showMember?: boolean;
    link?: boolean;
    deleted?: boolean;
    keyword?: string;
  }>(),
  {
    showIcon: true,
    showEmail: true,
    role: undefined,
    link: true,
    deleted: false,
    showMember: true,
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
