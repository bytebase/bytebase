<template>
  <NButton type="primary" size="large" @click="toExportCenter">
    <template #icon>
      <heroicons-outline:download class="w-5 h-5" />
    </template>

    {{ $t("export-center.self") }}
  </NButton>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";

import { useIssueContext } from "@/components/IssueV1";

const router = useRouter();
const { issue } = useIssueContext();

const toExportCenter = () => {
  router.push({
    name: "workspace.export-center",
    hash: `#${issue.value.uid}`,
  });
};
</script>
