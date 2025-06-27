<template>
  <div class="flex flex-row items-center space-x-2">
    <UserAvatar :user="user" />

    <div class="flex flex-row items-center">
      <div class="flex flex-col">
        <div class="flex flex-row items-center space-x-2">
          <div :class="convertStateToNew(user.state) === State.DELETED ? 'line-through' : ''">
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
          </div>
          <NTag
            v-if="convertStateToNew(user.state) === State.DELETED"
            size="small"
            round
            type="error"
          >
            {{ $t("settings.members.inactive") }}
          </NTag>
          <NTag v-if="user.profile?.source" size="small" round type="primary">
            {{ user.profile.source }}
          </NTag>
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
import { NTag } from "naive-ui";
import { computed } from "vue";
import UserAvatar from "@/components/User/UserAvatar.vue";
import ServiceAccountTag from "@/components/misc/ServiceAccountTag.vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import { useCurrentUserV1, usePermissionStore } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { unknownUser } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { convertStateToNew } from "@/utils/v1/common-conversions";
import { User, UserType } from "@/types/proto/v1/user_service";
import type { MemberBinding } from "../../types";

const props = defineProps<{
  binding: MemberBinding;
  onClickUser?: (user: User, event: MouseEvent) => void;
}>();

const currentUserV1 = useCurrentUserV1();
const permissionStore = usePermissionStore();

const user = computed(() => props.binding.user ?? unknownUser());
</script>
