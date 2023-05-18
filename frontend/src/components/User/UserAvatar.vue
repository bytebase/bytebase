<template>
  <BBAvatar :username="username" :size="size" />
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { BBAvatarSizeType } from "@/bbkit/types";
import { UNKNOWN_ID, unknownUser } from "@/types";
import { User } from "@/types/proto/v1/auth_service";

const props = defineProps({
  user: {
    type: Object as PropType<User>,
    default: () => unknownUser(),
  },
  size: {
    type: String as PropType<BBAvatarSizeType>,
    default: "NORMAL",
  },
});

const username = computed((): string => {
  if (props.user.name === `users/${UNKNOWN_ID}`) {
    return "?";
  }
  return props.user.title;
});
</script>
