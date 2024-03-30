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
import type { PropType } from "vue";
import { computed } from "vue";
import { BBAvatar } from "@/bbkit";
import type { BBAvatarSizeType } from "@/bbkit/types";
import { UNKNOWN_ID, unknownUser } from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import type { VueClass } from "@/utils";

const props = defineProps({
  user: {
    type: Object as PropType<User>,
    default: () => unknownUser(),
  },
  size: {
    type: String as PropType<BBAvatarSizeType>,
    default: "NORMAL",
  },
  overrideClass: {
    type: [String, Object, Array] as PropType<VueClass>,
    default: undefined,
  },
  overrideTextSize: {
    type: String,
    default: undefined,
  },
});

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
