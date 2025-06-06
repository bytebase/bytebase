<template>
  <NTag size="small" round :type="tagType">
    <template #icon>
      <component :is="statusIcon" class="w-3 h-3" />
    </template>
    {{ statusText }}
  </NTag>
</template>

<script setup lang="ts">
import { CheckCircleIcon, AlertCircleIcon, XCircleIcon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed } from "vue";

const props = defineProps<{
  status: "SUCCESS" | "WARNING" | "ERROR" | "STATUS_UNSPECIFIED";
}>();

const statusIcon = computed(() => {
  switch (props.status) {
    case "SUCCESS":
      return CheckCircleIcon;
    case "WARNING":
      return AlertCircleIcon;
    case "ERROR":
      return XCircleIcon;
    default:
      return null;
  }
});

const statusText = computed(() => {
  switch (props.status) {
    case "SUCCESS":
      return "Pass";
    case "WARNING":
      return "Warn";
    case "ERROR":
      return "Error";
    default:
      return "";
  }
});

const tagType = computed(() => {
  switch (props.status) {
    case "SUCCESS":
      return "success";
    case "WARNING":
      return "warning";
    case "ERROR":
      return "error";
    default:
      return "default";
  }
});
</script>
