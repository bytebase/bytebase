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
        <NButton
          v-if="user.serviceKey"
          tertiary
          size="small"
          @click.prevent="() => copyServiceKey(user.serviceKey)"
        >
          <template #icon>
            <heroicons-outline:clipboard class="w-4 h-4" />
          </template>
          {{ $t("settings.members.copy-service-key") }}
        </NButton>
        <NButton
          v-else
          tertiary
          size="tiny"
          @click.prevent="$emit('reset-service-key', user)"
        >
          <template #icon>
            <heroicons-outline:reply class="w-4 h-4" />
          </template>
          {{ $t("settings.members.reset-service-key") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useCurrentUserV1 } from "@/store";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { UserType } from "@/types/proto/v1/auth_service";
import { hasWorkspacePermissionV2, toClipboard } from "@/utils";

defineProps<{
  user: User;
}>();

defineEmits<{
  (event: "reset-service-key", user: User): void;
}>();

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();

const allowEdit = computed(() => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.update");
});

const copyServiceKey = (serviceKey: string) => {
  toClipboard(serviceKey).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("settings.members.service-key-copied"),
    });
  });
};
</script>
