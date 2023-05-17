<template>
  <div v-if="descriptionType === 'TEXT'">
    {{ description }}
  </div>
  <div
    v-else-if="descriptionType === 'ISSUE'"
    class="normal-link"
    @click="gotoIssuePage"
  >
    {{ `#${issueId}` }}
  </div>
</template>

<script lang="ts" setup>
import { computed, defineProps, onMounted, ref } from "vue";
import { pushNotification, useIssueStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { issueSlug } from "@/utils";

type DescriptionType = "TEXT" | "ISSUE";

const issueDescriptionRegexp = /^#(\d+)$/;

const props = defineProps({
  description: {
    type: String,
    required: true,
  },
});

const issueId = ref(UNKNOWN_ID);

const descriptionType = computed<DescriptionType>(() => {
  if (issueDescriptionRegexp.test(props.description)) {
    return "ISSUE";
  }
  return "TEXT";
});

const gotoIssuePage = async () => {
  const issue = await useIssueStore().getOrFetchIssueById(issueId.value);
  if (issue.id === UNKNOWN_ID) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Issue #${issueId.value} not found`,
    });
    return;
  }

  window.open(`/issue/${issueSlug(issue.name, issue.id)}`, "_blank");
};

onMounted(() => {
  if (descriptionType.value === "ISSUE") {
    const match = props.description.match(issueDescriptionRegexp);
    if (match) {
      issueId.value = Number(match[1]);
    }
  }
});
</script>
