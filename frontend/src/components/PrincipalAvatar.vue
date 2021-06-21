<template>
  <BBAvatar :username="username" :size="size" />
</template>

<script lang="ts">
import { computed, PropType } from "@vue/runtime-core";
import { BBAvatarSizeType } from "../bbkit/types";
import { Principal, UNKNOWN_ID } from "../types";

export default {
  name: "PrincipalAvatar",
  props: {
    principal: {
      type: Object as PropType<Principal>,
      default: "",
    },
    size: {
      type: String as PropType<BBAvatarSizeType>,
      default: "NORMAL",
    },
  },
  setup(props, ctx) {
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
