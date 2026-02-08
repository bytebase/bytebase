<template>
  <UserLink v-if="user" :title="user.title" :email="user.email" />
  <span v-else class="font-medium text-error">{{ $t("common.unknown") }}</span>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { UserLink } from "@/components/v2/Model/cells";
import { useUserStore } from "@/store";

const props = defineProps<{
  // Format: users/{email}
  // Issue comments always have a real user creator
  creator: string;
}>();

const userStore = useUserStore();

const user = computedAsync(() => {
  return userStore.getOrFetchUserByIdentifier({
    identifier: props.creator,
  });
});
</script>
