<template>
  <component
    :is="isLink ? 'router-link' : 'span'"
    v-if="userEmail !== SYSTEM_BOT_EMAIL"
    v-bind="bindings"
    class="font-medium text-main whitespace-nowrap"
    :class="[isLink && 'hover:underline']"
  >
    {{ user?.title }}
  </component>
  <div v-else class="inline-flex items-center">
    <span class="font-medium text-main whitespace-nowrap">
      {{ user?.title }}
    </span>
    <span
      class="ml-0.5 inline-flex items-center px-1 py-0.5 rounded-lg text-xs font-semibold bg-green-100 text-green-800"
    >
      {{ $t("settings.members.system-bot") }}
    </span>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { usePageMode, useUserStore } from "@/store";
import { SYSTEM_BOT_EMAIL } from "@/types";
import { LogEntity } from "@/types/proto/v1/logging_service";
import { extractUserResourceName } from "@/utils";

const props = defineProps<{
  activity: LogEntity;
}>();

const pageMode = usePageMode();

const userEmail = computed(() => {
  return extractUserResourceName(props.activity.creator);
});

const user = computed(() => {
  return useUserStore().getUserByEmail(userEmail.value);
});

const isLink = computed(() => {
  return pageMode.value === "BUNDLED";
});

const bindings = computed(() => {
  if (isLink.value) {
    return {
      to: `/users/${userEmail.value}`,
      activeClass: "",
      exactActiveClass: "",
      onClick: (e: MouseEvent) => {
        e.stopPropagation();
      },
    };
  }
  return {};
});
</script>
