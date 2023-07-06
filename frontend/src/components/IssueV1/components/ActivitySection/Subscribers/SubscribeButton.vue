<template>
  <NButton text :disabled="isLoading" @click="toggleSubscribe">
    <BBSpin v-if="isLoading" class="w-4 h-4 mr-1" />
    <span>
      {{ isSubscribed ? $t("issue.unsubscribe") : $t("issue.subscribe") }}
    </span>
  </NButton>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { NButton } from "naive-ui";
import { pull } from "lodash-es";

import { useIssueContext } from "@/components/IssueV1";
import { useCurrentUserV1 } from "@/store";

const { issue } = useIssueContext();
const currentUser = useCurrentUserV1();
const isLoading = ref(false);

const isSubscribed = computed(() => {
  return issue.value.subscribers.includes(`users/${currentUser.value.email}`);
});

const toggleSubscribe = async () => {
  // TODO
  try {
    isLoading.value = true;
    await new Promise((resolve) => setTimeout(resolve, 500));
    if (isSubscribed.value) {
      pull(issue.value.subscribers, `users/${currentUser.value.email}`);
    } else {
      issue.value.subscribers.push(`users/${currentUser.value.email}`);
    }
  } finally {
    isLoading.value = false;
  }
};
</script>
