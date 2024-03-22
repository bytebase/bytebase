<template>
  <img :class="['h-4 w-auto', customClass]" :src="icon" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";

const props = defineProps<{
  customClass?: string;
  type: VCSProvider_Type;
}>();

const icon = computed(() => {
  let iconType = "";
  switch (props.type) {
    case VCSProvider_Type.AZURE_DEVOPS:
      iconType = "azure-devops";
      break;
    case VCSProvider_Type.GITLAB:
      iconType = "gitlab";
      break;
    case VCSProvider_Type.GITHUB:
      iconType = "github";
      break;
    case VCSProvider_Type.BITBUCKET:
      iconType = "bitbucket";
      break;
    default:
      iconType = "";
  }
  if (iconType === "") {
    return iconType;
  }
  return new URL(`../../assets/${iconType}-logo.svg`, import.meta.url).href;
});
</script>
