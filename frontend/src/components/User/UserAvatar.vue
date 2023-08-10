<template>
  <BBAvatar
    :username="username"
    :size="size"
    :override-class="overrideClass"
    :override-text-size="overrideTextSize"
  />
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { BBAvatar } from "@/bbkit";
import { BBAvatarSizeType } from "@/bbkit/types";
import { UNKNOWN_ID, unknownUser } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { VueClass } from "@/utils";

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
</script>
