<template>
  <BBAvatar :username="username" :size="size" />
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { BBAvatarSizeType } from "../bbkit/types";
import { Principal, unknown, UNKNOWN_ID } from "../types";

export default defineComponent({
  name: "PrincipalAvatar",
  props: {
    principal: {
      type: Object as PropType<Principal>,
      default: () => unknown("PRINCIPAL"),
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
});
</script>
