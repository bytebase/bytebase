<template>
  <component
    :is="isLink ? 'router-link' : 'div'"
    :class="isLink && 'normal-link'"
    :to="{
      name: WORKSPACE_ROUTE_USER_PROFILE,
      params: {
        principalEmail: email,
      },
    }"
  >
    <HighlightLabelText
      :keyword="keyword ?? ''"
      :text="title"
    />
  </component>
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import { HighlightLabelText } from "@/components/v2";
import { WORKSPACE_ROUTE_USER_PROFILE } from "@/router/dashboard/workspaceRoutes";
import { getUserTypeByEmail } from "@/types";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = withDefaults(
  defineProps<{
    email: string;
    title: string;
    keyword?: string;
    link?: boolean;
  }>(),
  {
    link: true,
  }
);

const isEndUser = computed(
  () => getUserTypeByEmail(props.email) === UserType.USER
);

const isLink = computed(
  () =>
    props.link && isEndUser.value && hasWorkspacePermissionV2("bb.users.get")
);
</script>