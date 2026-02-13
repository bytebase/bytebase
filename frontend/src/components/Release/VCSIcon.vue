<template>
  <img :class="['h-4 w-auto', customClass]" :src="icon" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { VCSType } from "@/types/proto-es/v1/common_pb";

const props = defineProps<{
  customClass?: string;
  type: VCSType;
}>();

const icon = computed(() => {
  let iconType = "";
  switch (props.type) {
    case VCSType.AZURE_DEVOPS:
      iconType = "azure-devops";
      break;
    case VCSType.GITLAB:
      iconType = "gitlab";
      break;
    case VCSType.GITHUB:
      iconType = "github";
      break;
    case VCSType.BITBUCKET:
      iconType = "bitbucket";
      break;
    default:
      iconType = "";
  }
  if (iconType === "") {
    return iconType;
  }
  return new URL(`../../assets/git/${iconType}.svg`, import.meta.url).href;
});
</script>
