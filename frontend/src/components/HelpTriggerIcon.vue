<template>
  <heroicons-outline:question-mark-circle
    class="w-5 h-5 cursor-pointer"
    @click="handleClick"
  />
</template>

<script lang="ts">
import { defineComponent, inject } from "vue";
import { Event } from "@/utils";
import { EventType } from "@/types";

export default defineComponent({
  name: "HelpTrigger",
  props: {
    name: {
      type: String,
      required: true,
      default: "",
    },
    isGuide: {
      type: Boolean,
      required: false,
      default: false,
    },
  },
  setup: (props) => {
    const event = inject("event") as Event;
    const handleClick = () => {
      event.emit(EventType.EVENT_HELP, props.name, props.isGuide);
    };
    return {
      handleClick,
    };
  },
});
</script>

<style></style>
