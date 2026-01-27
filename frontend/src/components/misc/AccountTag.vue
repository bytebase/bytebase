<template>
  <NTag v-if="user.userType === UserType.SERVICE_ACCOUNT || user.userType === UserType.WORKLOAD_IDENTITY" v-bind="$attrs" round type="info">
    {{ user.userType === UserType.SERVICE_ACCOUNT
      ? $t("settings.members.service-account")
      : $t("settings.members.workload-identity")
    }}
  </NTag>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { type User, UserType } from "@/types/proto-es/v1/user_service_pb";

defineProps<{
  user: User;
}>();
</script>
