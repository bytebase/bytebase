<template>
  <NButton text :disabled="isLoading" @click="handleClick">
    <BBSpin v-if="isLoading" class="w-4 h-4 mr-1" />
    <span>
      {{ isSubscribed ? $t("issue.unsubscribe") : $t("issue.subscribe") }}
    </span>
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useIssueContext, toggleSubscribeIssue } from "@/components/IssueV1";
import { useCurrentUserV1 } from "@/store";

const { issue } = useIssueContext();
const currentUser = useCurrentUserV1();
const isLoading = ref(false);

const isSubscribed = computed(() => {
  return issue.value.subscribers.includes(`users/${currentUser.value.email}`);
});

const handleClick = async () => {
  // TODO
  try {
    isLoading.value = true;
    await toggleSubscribeIssue(issue.value, currentUser.value);
  } finally {
    isLoading.value = false;
  }
};
</script>
