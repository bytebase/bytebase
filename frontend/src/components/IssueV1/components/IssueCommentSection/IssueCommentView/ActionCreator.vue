<template>
  <component
    :is="!disallowNavigateToConsole ? 'router-link' : 'span'"
    v-if="userEmail !== userStore.systemBotUser?.email"
    v-bind="bindings"
    class="font-medium text-main whitespace-nowrap"
    :class="[!disallowNavigateToConsole && 'hover:underline']"
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
import { computed } from "vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import { useAppFeature, useUserStore } from "@/store";
import { extractUserResourceName } from "@/utils";

const props = defineProps<{
  // Format: users/{email}
  creator: string;
}>();

const userStore = useUserStore();
const disallowNavigateToConsole = useAppFeature(
  "bb.feature.disallow-navigate-to-console"
);

const userEmail = computed(() => {
  return extractUserResourceName(props.creator);
});

const user = computed(() => {
  return userStore.getUserByEmail(userEmail.value);
});

const bindings = computed(() => {
  if (disallowNavigateToConsole.value) {
    return undefined;
  }
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
