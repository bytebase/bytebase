<template>
  <heroicons-outline:question-mark-circle
    class="w-5 h-5 cursor-pointer"
    @click="handleClick"
  />
</template>

<script lang="ts">
import { defineComponent, inject } from "vue";
import { Event } from "@/utils";

export default defineComponent({
  name: "HelpTrigger",
  props: {
    name: {
      type: String,
      required: true,
      default: "",
    },
  },
  setup: (props) => {
    const event = inject("event") as Event;
    const handleClick = () => {
      event.emit("help", props.name);
    };
    return {
      handleClick,
    };
  },
});
</script>

<style></style>
