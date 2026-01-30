<template>
  <UserLink v-if="user" :title="user.title" :email="user.email" />
  <span v-else class="font-medium text-main whitespace-nowrap">{{ $t("common.system") }}</span>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { UserLink } from "@/components/v2/Model/cells";
import { useUserStore } from "@/store";

const props = defineProps<{
  // Format: users/{email}
  creator: string;
}>();

const userStore = useUserStore();

const user = computedAsync(() => {
  if (!props.creator) return undefined;
  return userStore.getOrFetchUserByIdentifier(props.creator);
});
</script>
