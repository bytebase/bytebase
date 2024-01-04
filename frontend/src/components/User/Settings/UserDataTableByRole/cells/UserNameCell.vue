<template>
  <div class="flex flex-row items-center shrink space-x-2">
    <UserAvatar :user="user" />

    <div class="flex flex-row items-center">
      <div class="flex flex-col">
        <div class="flex flex-row items-center space-x-2">
          <router-link :to="`/users/${user.email}`" class="normal-link">
            {{ user.title }}
          </router-link>
          <span
            v-if="currentUserV1.name === user.name"
            class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
          >
            {{ $t("settings.members.yourself") }}
          </span>
          <span
            v-if="user.name === SYSTEM_BOT_USER_NAME"
            class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
          >
            {{ $t("settings.members.system-bot") }}
          </span>
          <span
            v-if="user.userType === UserType.SERVICE_ACCOUNT"
            class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
          >
            {{ $t("settings.members.service-account") }}
          </span>
        </div>
        <span v-if="user.name !== SYSTEM_BOT_USER_NAME" class="textlabel">
          {{ user.email }}
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useCurrentUserV1 } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import { User, UserType } from "@/types/proto/v1/auth_service";

defineProps<{
  user: User;
}>();

defineEmits<{
  (event: "reset-service-key", user: User): void;
}>();

const currentUserV1 = useCurrentUserV1();
</script>
