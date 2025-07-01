<template>
  <div class="flex flex-row items-center shrink space-x-2">
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
          <router-link v-else :to="`/users/${user.email}`" class="normal-link">
            {{ user.title }}
          </router-link>
          <YouTag v-if="currentUserV1.name === user.name" />
          <SystemBotTag v-if="user.name === SYSTEM_BOT_USER_NAME" />
          <ServiceAccountTag
            v-if="user.userType === UserType.SERVICE_ACCOUNT"
          />
          <NTag
            v-if="role"
            size="small"
            round
            :type="role === GroupMember_Role.OWNER ? 'primary' : 'default'"
          >
            {{
              (() => {
                switch(role) {
                  case GroupMember_Role.OWNER:
                    return $t('settings.members.groups.form.role.owner');
                  case GroupMember_Role.MEMBER:
                    return $t('settings.members.groups.form.role.member');
                  default:
                    return 'ROLE UNRECOGNIZED';
                }
              })()
            }}
          </NTag>
        </div>
        <span v-if="user.name !== SYSTEM_BOT_USER_NAME" class="textlabel">
          {{ user.email }}
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import UserAvatar from "@/components/User/UserAvatar.vue";
import ServiceAccountTag from "@/components/misc/ServiceAccountTag.vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import { useCurrentUserV1, usePermissionStore } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { UserType, type User } from "@/types/proto-es/v1/user_service_pb";
import {
  GroupMember_Role,
} from "@/types/proto-es/v1/group_service_pb";

withDefaults(
  defineProps<{
    user: User;
    role?: GroupMember_Role;
    onClickUser?: (user: User, event: MouseEvent) => void;
  }>(),
  {
    role: undefined,
    onClickUser: undefined,
  }
);

defineEmits<{
  (event: "reset-service-key", user: User): void;
}>();

const currentUserV1 = useCurrentUserV1();
const permissionStore = usePermissionStore();
</script>
