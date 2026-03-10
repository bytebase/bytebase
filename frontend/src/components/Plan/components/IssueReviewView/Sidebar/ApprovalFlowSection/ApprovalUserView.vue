<template>
  <UserNameCell
    v-if="candidateUser"
    :user="candidateUser"
    size="tiny"
    :show-mfa-enabled="false"
    :show-source="false"
    :allow-edit="false"
    :show-email="false"
  />
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { UserNameCell } from "@/components/v2/Model/cells";
import { useUserStore } from "@/store";
import { AccountType, getAccountTypeByEmail } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";

const props = defineProps<{
  // candidate in users/{email} format.
  candidate: string;
}>();

const userStore = useUserStore();

const candidateUser = computedAsync(async () => {
  const user = await userStore.getOrFetchUserByIdentifier({
    identifier: props.candidate,
  });
  if (!user) {
    return;
  }
  if (
    getAccountTypeByEmail(user.email) !== AccountType.USER ||
    user.state !== State.ACTIVE
  ) {
    return;
  }
  return user;
});
</script>
