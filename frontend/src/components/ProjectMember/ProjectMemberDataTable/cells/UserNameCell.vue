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
import { useCurrentUserV1 } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { UserType } from "@/types/proto/v1/auth_service";
import type { ProjectMember } from "../../types";

const props = defineProps<{
  projectMember: ProjectMember;
}>();

defineEmits<{
  (event: "reset-service-key", user: User): void;
}>();

const currentUserV1 = useCurrentUserV1();

const user = computed(() => props.projectMember.user);
</script>
