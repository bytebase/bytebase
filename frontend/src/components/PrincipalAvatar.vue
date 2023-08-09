<template>
  <BBAvatar
    :username="name"
    :size="size"
    :override-class="overrideClass"
    :override-text-size="overrideTextSize"
  />
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { BBAvatar } from "@/bbkit";
import { BBAvatarSizeType } from "@/bbkit/types";
import { Principal, unknown, UNKNOWN_ID } from "@/types";
import { VueClass } from "@/utils";

export default defineComponent({
  name: "PrincipalAvatar",
  components: { BBAvatar },
  props: {
    principal: {
      type: Object as PropType<Principal>,
      default: () => unknown("PRINCIPAL"),
    },
    username: {
      type: String,
      default: "?",
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
  },
  setup(props) {
    const name = computed((): string => {
      if (props.principal.id == UNKNOWN_ID) {
        return props.username;
      }
      return props.principal.name;
    });
    return { name };
  },
});
</script>
