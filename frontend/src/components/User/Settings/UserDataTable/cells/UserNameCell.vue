<template>
  <div class="flex flex-row items-center space-x-2">
    <UserAvatar :user="user" />

    <div class="flex flex-row items-center">
      <div class="flex flex-col">
        <div class="flex flex-row items-center space-x-2">
          <span
            v-if="onClickUser"
            class="normal-link truncate max-w-[10rem]"
            @click="onClickUser(user, $event)"
          >
            {{ user.title }}
          </span>
          <span
            v-else-if="permissionStore.onlyWorkspaceMember"
            class="truncate max-w-[10em]"
          >
            {{ user.title }}
          </span>
          <router-link
            v-else
            :to="`/users/${user.email}`"
            class="normal-link truncate max-w-[10em]"
          >
            {{ user.title }}
          </router-link>
          <NTag v-if="user.profile?.source" size="small" round type="primary">
            {{ user.profile.source }}
          </NTag>
          <YouTag v-if="currentUserV1.name === user.name" />
          <SystemBotTag v-if="user.name === SYSTEM_BOT_USER_NAME" />
          <ServiceAccountTag
            v-if="user.userType === UserType.SERVICE_ACCOUNT"
          />
          <span
            v-if="user.mfaEnabled"
            class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs bg-green-800 text-green-100"
          >
            {{ $t("two-factor.enabled") }}
          </span>
        </div>
        <span v-if="user.name !== SYSTEM_BOT_USER_NAME" class="textlabel">
          {{ user.email }}
        </span>
      </div>
      <div
        v-if="user.userType === UserType.SERVICE_ACCOUNT && allowEdit"
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
import { NButton, NTag } from "naive-ui";
import { computed } from "vue";
import UserAvatar from "@/components/User/UserAvatar.vue";
import ServiceAccountTag from "@/components/misc/ServiceAccountTag.vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import { CopyButton } from "@/components/v2";
import { useCurrentUserV1, usePermissionStore } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { UserType, type User } from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

defineProps<{
  user: User;
  onClickUser?: (user: User, event: MouseEvent) => void;
}>();

defineEmits<{
  (event: "reset-service-key", user: User): void;
}>();

const currentUserV1 = useCurrentUserV1();
const permissionStore = usePermissionStore();

const allowEdit = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});
</script>
