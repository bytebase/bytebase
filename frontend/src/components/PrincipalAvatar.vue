<template>
  <BBAvatar :username="username" :size="size" />
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { BBAvatarSizeType } from "../bbkit/types";
import { Principal, UNKNOWN_ID } from "../types";

export default {
  name: "PrincipalAvatar",
  props: {
    principal: {
      type: Object as PropType<Principal>,
      default: "", // This line raises an eslint ERROR. But I won't change it this time.
    },
    size: {
      type: String as PropType<BBAvatarSizeType>,
      default: "NORMAL",
    },
  },
  setup(props) {
    const username = computed((): string => {
      if (props.principal.id == UNKNOWN_ID) {
        return "?";
      }
      return props.principal.name;
    });
    return { username };
  },
};
</script>
