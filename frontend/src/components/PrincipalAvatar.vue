<template>
  <BBAvatar :username="principal.name" :size="size" />
</template>

<script lang="ts">
import { PropType } from "@vue/runtime-core";
import { BBAvatarSizeType } from "../bbkit/types";
import { Principal } from "../types";
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
    return {};
  },
};
</script>
