<template>
  <div class="flex flex-row items-center" :class="gapClass">
    <UserAvatar :user="user" :size="avatarSize" />

    <div class="flex flex-row items-center">
      <div class="flex flex-col">
        <div class="flex flex-row items-center gap-x-1">
          <div :class="isDeleted ? 'line-through' : ''">
            <HighlightLabelText
              v-if="onClickUser"
              class="truncate max-w-40"
              :keyword="keyword"
              :text="user.title"
              @click="onClickUser(user, $event)"
            />
            <UserLink
              v-else
              :keyword="keyword"
              :title="user.title"
              :email="user.email"
              :link="link"
            />
          </div>
          <slot name="suffix">
          </slot>
          <NTag v-if="isDeleted" :size="tagSize" type="error" round>
            {{$t("common.deleted")}}
          </NTag>
          <NTag v-if="user.profile?.source && showSource" :size="tagSize" round type="primary">
            {{ user.profile.source }}
          </NTag>
          <YouTag v-if="currentUserV1.name === user.name" :size="tagSize"/>
          <SystemBotTag v-if="user.name === SYSTEM_BOT_USER_NAME" :size="tagSize"/>
          <ServiceAccountTag
            v-if="user.userType === UserType.SERVICE_ACCOUNT"
            :size="tagSize"
          />
          <WorkloadIdentityTag
            v-if="user.userType === UserType.WORKLOAD_IDENTITY"
            :size="tagSize"
          />
          <NTag v-if="user.mfaEnabled && showMfaEnabled" :size="tagSize" type="success" round>
            {{ $t("two-factor.enabled") }}
          </NTag>
        </div>
        <slot name="footer">
          <NEllipsis
            v-if="user.name !== SYSTEM_BOT_USER_NAME && showEmail"
            class="textinfolabel"
            :line-clamp="1"
            :tooltip="true"
          >
            {{ user.email }}
          </NEllipsis>
        </slot>
      </div>
      <div
        v-if="!isDeleted && user.userType === UserType.SERVICE_ACCOUNT && allowEdit && hasPermission"
        class="ml-3 text-xs"
      >
        <CopyButton
          v-if="user.serviceKey"
          quaternary
          size="small"
          :text="false"
          :tertiary="true"
          :content="user.serviceKey"
        >
          {{ $t("settings.members.copy-service-key") }}
        </CopyButton>
        <NButton
          v-else
          tertiary
          size="small"
          @click.prevent="$emit('reset-service-key', user)"
        >
          <template #icon>
            <ReplyIcon class="w-4 h-4" />
          </template>
          {{ $t("settings.members.reset-service-key") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ReplyIcon } from "lucide-vue-next";
import { NButton, NEllipsis, NTag } from "naive-ui";
import { computed } from "vue";
import ServiceAccountTag from "@/components/misc/ServiceAccountTag.vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import WorkloadIdentityTag from "@/components/misc/WorkloadIdentityTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { CopyButton, HighlightLabelText } from "@/components/v2";
import { useCurrentUserV1 } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import UserLink from "./UserLink.vue";

const props = withDefaults(
  defineProps<{
    user: User;
    allowEdit?: boolean;
    keyword?: string;
    size?: "tiny" | "small" | "medium";
    link?: boolean;
    showSource?: boolean;
    showMfaEnabled?: boolean;
    showEmail?: boolean;
    onClickUser?: (user: User, event: MouseEvent) => void;
  }>(),
  {
    allowEdit: true,
    size: "medium",
    link: true,
    showSource: true,
    showMfaEnabled: true,
    showEmail: true,
  }
);

defineEmits<{
  (event: "reset-service-key", user: User): void;
}>();

const currentUserV1 = useCurrentUserV1();

const hasPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

const isDeleted = computed(() => props.user.state === State.DELETED);

const tagSize = computed(() => {
  if (props.size === "tiny") {
    return "tiny";
  }
  return "small";
});

const avatarSize = computed(() => {
  switch (props.size) {
    case "tiny":
      return "TINY";
    case "small":
      return "SMALL";
    default:
      return "NORMAL";
  }
});

const gapClass = computed(() => {
  if (props.size === "tiny") {
    return "gap-x-1";
  }
  return "gap-x-2";
});
</script>
