<template>
  <HelpCircleIcon class="w-4 h-4 cursor-pointer" @click="handleClick" />
</template>

<script lang="ts" setup>
import { HelpCircleIcon } from "lucide-vue-next";
import { useHelpStore } from "@/store";

const props = defineProps<{
  id: string;
  openByDefault?: boolean;
}>();

const helpStore = useHelpStore();
const handleClick = () => {
  helpStore.showHelp(props.id, props.openByDefault);
};
</script>
