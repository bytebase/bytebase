<template>
  <div class="flex flex-row items-center space-x-2">
    <UserAvatar :user="user" />

    <div class="flex flex-row items-center">
      <div class="flex flex-col">
        <div class="flex flex-row items-center space-x-2">
          <router-link
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
import { useCurrentUserV1, useUserStore } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { unknownUser } from "@/types";
import { UserType } from "@/types/proto/v1/auth_service";
import type { ProjectBinding } from "../../types";

const props = defineProps<{
  projectMember: ProjectBinding;
}>();

const currentUserV1 = useCurrentUserV1();
const userStore = useUserStore();

const user = computed(
  () => userStore.getUserByEmail(props.projectMember.email) ?? unknownUser()
);
</script>
