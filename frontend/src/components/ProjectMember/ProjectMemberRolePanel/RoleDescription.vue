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
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { issueServiceClient } from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { pushNotification } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { extractProjectResourceName, issueSlug } from "@/utils";

type DescriptionType = "TEXT" | "ISSUE";

const issueDescriptionRegexp = /^#(\d+)$/;

const props = defineProps({
  description: {
    type: String,
    required: true,
  },
});

const router = useRouter();
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

  const route = router.resolve({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(issue.name),
      issueSlug: issueSlug(issue.title, issue.uid),
    },
  });
  window.open(route.href, "_blank");
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
