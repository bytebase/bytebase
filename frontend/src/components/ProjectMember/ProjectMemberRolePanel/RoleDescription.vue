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
import { create } from "@bufbuild/protobuf";
import { issueServiceClientConnect } from "@/grpcweb";
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { pushNotification } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { isValidIssueName } from "@/utils";

type DescriptionType = "TEXT" | "ISSUE";

const issueDescriptionRegexp = /^#(\d+)$/;

const props = defineProps<{
  description: string;
}>();

const router = useRouter();
const issueUID = ref(String(UNKNOWN_ID));

const descriptionType = computed<DescriptionType>(() => {
  if (issueDescriptionRegexp.test(props.description)) {
    return "ISSUE";
  }
  return "TEXT";
});

const gotoIssuePage = async () => {
  const request = create(GetIssueRequestSchema, {
    name: `projects/-/issues/${issueUID.value}`,
  });
  const issue = await issueServiceClientConnect.getIssue(request);
  if (!isValidIssueName(issue.name)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Issue #${issueUID.value} not found`,
    });
    return;
  }

  const route = router.resolve({
    path: `/${issue.name}`,
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
