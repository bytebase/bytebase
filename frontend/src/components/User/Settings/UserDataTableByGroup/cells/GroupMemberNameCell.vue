<template>
  <div class="flex flex-row items-center shrink space-x-2">
    <UserAvatar :user="user" />

    <div class="flex flex-row items-center">
      <div class="flex flex-col">
        <div class="flex flex-row items-center space-x-2">
          <router-link :to="`/users/${user.email}`" class="normal-link">
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
            :type="role === UserGroupMember_Role.OWNER ? 'primary' : 'default'"
          >
            {{
              $t(
                `settings.members.groups.form.role.${userGroupMember_RoleToJSON(role).toLowerCase()}`
              )
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
import { useCurrentUserV1 } from "@/store";
import { SYSTEM_BOT_USER_NAME, type ComposedUser } from "@/types";
import { UserType } from "@/types/proto/v1/auth_service";
import {
  UserGroupMember_Role,
  userGroupMember_RoleToJSON,
} from "@/types/proto/v1/user_group";

withDefaults(
  defineProps<{
    user: ComposedUser;
    role?: UserGroupMember_Role;
  }>(),
  {
    role: undefined,
  }
);

defineEmits<{
  (event: "reset-service-key", user: ComposedUser): void;
}>();

const currentUserV1 = useCurrentUserV1();
</script>
