<template>
  <div v-if="candidateUser" class="inline-flex items-center gap-1">
    <UserAvatar :user="candidateUser" :size="avatarSize" />
    <span class="text-xs font-medium">{{ candidateUser.title }}</span>
    <NTag
      v-if="currentUser.name === candidateUser.name"
      size="tiny"
      round
      type="success"
    >
      {{ $t("custom-approval.issue-review.you") }}
    </NTag>
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { NTag } from "naive-ui";
import { computed } from "vue";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { useCurrentUserV1, useUserStore } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import { UserType } from "@/types/proto/v1/user_service";
import { convertStateToOld } from "@/utils/v1/common-conversions";

type SizeType = "tiny" | "small" | "normal";

const props = withDefaults(
  defineProps<{
    // candidate in users/{email} format.
    candidate: string;
    size?: SizeType;
  }>(),
  {
    size: "normal",
  }
);

const currentUser = useCurrentUserV1();
const userStore = useUserStore();

const avatarSize = computed(() => {
  switch (props.size) {
    case "tiny":
      return "TINY";
    case "small":
      return "SMALL";
    default:
      return "NORMAL";
  }
});

const candidateUser = computedAsync(async () => {
  const user = await userStore.getOrFetchUserByIdentifier(props.candidate);
  if (!user) {
    return;
  }
  if (user.userType !== UserType.USER || user.state !== convertStateToOld(State.ACTIVE)) {
    return;
  }
  return user;
});
</script>
