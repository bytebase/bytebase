<template>
  <div class="inline-flex items-center gap-1">
    <UserAvatar :user="user" :size="avatarSize" />
    <span v-if="showName" :class="textClass">
      {{ displayName }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { BBAvatarSizeType } from "@/bbkit/types";
import UserAvatar from "./UserAvatar.vue";

type SizeType = "tiny" | "small" | "normal" | "large";

const props = withDefaults(
  defineProps<{
    user: User;
    size?: SizeType;
    showName?: boolean;
    nameOnly?: boolean;
  }>(),
  {
    size: "normal",
    showName: true,
    nameOnly: false,
  }
);

const displayName = computed(() => {
  if (props.nameOnly) {
    return props.user.title || props.user.email.split("@")[0];
  }
  return props.user.title || props.user.email;
});

const avatarSize = computed((): BBAvatarSizeType => {
  const sizeMap: Record<SizeType, BBAvatarSizeType> = {
    tiny: "TINY",
    small: "SMALL",
    normal: "NORMAL",
    large: "LARGE",
  };
  return sizeMap[props.size];
});

const textClass = computed(() => {
  const sizeClasses: Record<SizeType, string> = {
    tiny: "text-xs",
    small: "text-sm",
    normal: "text-base",
    large: "text-lg",
  };
  return sizeClasses[props.size];
});
</script>