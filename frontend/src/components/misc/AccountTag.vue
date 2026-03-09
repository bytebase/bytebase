<template>
  <NTag v-if="accountType === AccountType.SERVICE_ACCOUNT || accountType === AccountType.WORKLOAD_IDENTITY" v-bind="$attrs" round type="info">
    {{ accountType === AccountType.SERVICE_ACCOUNT
      ? $t("settings.members.service-account")
      : $t("settings.members.workload-identity")
    }}
  </NTag>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { computed } from "vue";
import { AccountType, getAccountTypeByEmail } from "@/types";
import type { User } from "@/types/proto-es/v1/user_service_pb";

const props = defineProps<{
  user: User;
}>();

const accountType = computed(() => getAccountTypeByEmail(props.user.email));
</script>
