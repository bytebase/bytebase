<template>
  <heroicons-outline:question-mark-circle
    class="w-5 h-5 cursor-pointer"
    @click="handleClick"
  />
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { useHelpStore } from "@/store";

export default defineComponent({
  name: "HelpTrigger",
  props: {
    id: {
      type: String,
      required: true,
      default: "",
    },
    openByDefault: {
      type: Boolean,
      required: false,
      default: false,
    },
  },
  setup: (props) => {
    const helpStore = useHelpStore();
    const handleClick = () => {
      helpStore.showHelp(props.id, props.openByDefault);
    };
    return {
      handleClick,
    };
  },
});
</script>
