<template>
  <div
    v-if="candidateUser"
    class="flex items-center py-1 gap-x-1"
    :class="[candidateUser.name === currentUser.name && 'font-bold']"
  >
    <UserAvatar :user="candidateUser" size="SMALL" />
    <span class="whitespace-nowrap">{{ candidateUser.title }}</span>
    <span class="whitespace-nowrap opacity-80">
      ({{ candidateUser.email }})
    </span>
    <span
      v-if="currentUser.name === candidateUser.name"
      class="inline-flex items-center px-1 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
    >
      {{ $t("custom-approval.issue-review.you") }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { useCurrentUserV1, useUserStore } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import { convertStateToNew } from "@/utils/v1/common-conversions";
import { UserType } from "@/types/proto/v1/user_service";

const props = defineProps<{
  // candidate in users/{email} format.
  candidate: string;
}>();

const currentUser = useCurrentUserV1();
const userStore = useUserStore();

const candidateUser = computedAsync(async () => {
  const user = await userStore.getOrFetchUserByIdentifier(props.candidate);
  if (!user) {
    return;
  }
  if (user.userType !== UserType.USER || convertStateToNew(user.state) !== State.ACTIVE) {
    return;
  }
  return user;
});
</script>
