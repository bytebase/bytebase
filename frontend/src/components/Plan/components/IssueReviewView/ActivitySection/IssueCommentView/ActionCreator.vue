<template>
  <component
    :is="'router-link'"
    v-if="userEmail !== userStore.systemBotUser?.email"
    v-bind="bindings"
    class="font-semibold text-gray-900 whitespace-nowrap hover:underline"
  >
    {{ user?.title }}
  </component>
  <div v-else class="inline-flex items-center gap-1">
    <span class="font-medium text-main whitespace-nowrap">
      {{ user?.title }}
    </span>
    <SystemBotTag />
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import { extractUserId, useUserStore } from "@/store";

const props = defineProps<{
  // Format: users/{email}
  creator: string;
}>();

const userStore = useUserStore();

const userEmail = computed(() => {
  return extractUserId(props.creator);
});

const user = computedAsync(() => {
  return userStore.getOrFetchUserByIdentifier(props.creator);
});

const bindings = computed(() => {
  return {
    to: `/users/${userEmail.value}`,
    activeClass: "",
    exactActiveClass: "",
    onClick: (e: MouseEvent) => {
      e.stopPropagation();
    },
  };
});
</script>
