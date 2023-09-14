<template>
  <div v-if="descriptionType === 'TEXT'">
    {{ description }}
  </div>
  <div
    v-else-if="descriptionType === 'ISSUE'"
    class="normal-link"
    @click="gotoIssuePage"
  >
    {{ `#${issueUID}` }}
  </div>
</template>

<script lang="ts" setup>
import { computed, defineProps, onMounted, ref } from "vue";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";
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

const issueUID = ref(String(UNKNOWN_ID));

const descriptionType = computed<DescriptionType>(() => {
  if (issueDescriptionRegexp.test(props.description)) {
    return "ISSUE";
  }
  return "TEXT";
});

const gotoIssuePage = async () => {
  const issue = await issueServiceClient.getIssue({
    name: `projects/-/issues/${issueUID.value}`,
  });
  if (issue.uid === String(UNKNOWN_ID)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Issue #${issueUID.value} not found`,
    });
    return;
  }

  window.open(`/issue/${issueSlug(issue.title, issue.uid)}`, "_blank");
};

onMounted(() => {
  if (descriptionType.value === "ISSUE") {
    const match = props.description.match(issueDescriptionRegexp);
    if (match) {
      issueUID.value = match[1];
    }
  }
});
</script>
