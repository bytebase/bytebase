<template>
  <BBAvatar
    :username="username"
    :email="email"
    :size="size"
    :override-class="overrideClass"
    :override-text-size="overrideTextSize"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { BBAvatar } from "@/bbkit";
import type { BBAvatarSizeType } from "@/bbkit/types";
import { UNKNOWN_ID, unknownUser } from "@/types";
import { type User } from "@/types/proto-es/v1/user_service_pb";
import type { VueClass } from "@/utils";

const props = withDefaults(
  defineProps<{
    user?: User;
    size?: BBAvatarSizeType;
    overrideClass?: VueClass;
    overrideTextSize?: string;
  }>(),
  {
    user: () => unknownUser(),
  }
);

const username = computed((): string => {
  if (props.user.name === `users/${UNKNOWN_ID}`) {
    return "?";
  }
  return props.user.title;
});
const email = computed(() => {
  if (props.user.name === `users/${UNKNOWN_ID}`) {
    return undefined;
  }
  return props.user.email;
});
</script>
