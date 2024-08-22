<template>
  <div class="flex flex-row items-center space-x-2">
    <UserAvatar :user="user" />

    <div class="flex flex-row items-center">
      <div class="flex flex-col">
        <div class="flex flex-row items-center space-x-2">
          <span
            class="truncate max-w-[10em]"
            v-if="permissionStore.onlyWorkspaceMember"
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
          <YouTag v-if="currentUserV1.name === user.name" />
          <SystemBotTag v-if="user.name === SYSTEM_BOT_USER_NAME" />
          <ServiceAccountTag
            v-if="user.userType === UserType.SERVICE_ACCOUNT"
          />
        </div>
        <span v-if="user.name !== SYSTEM_BOT_USER_NAME" class="textlabel">
          {{ user.email }}
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import UserAvatar from "@/components/User/UserAvatar.vue";
import ServiceAccountTag from "@/components/misc/ServiceAccountTag.vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import { useCurrentUserV1, usePermissionStore } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { unknownUser } from "@/types";
import { UserType } from "@/types/proto/v1/auth_service";
import type { MemberBinding } from "../../types";

const props = defineProps<{
  binding: MemberBinding;
}>();

const currentUserV1 = useCurrentUserV1();
const permissionStore = usePermissionStore();

const user = computed(() => props.binding.user ?? unknownUser());
</script>
