<template>
  <img :class="['h-4 w-auto', customClass]" :src="icon" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { ExternalVersionControl_Type } from "@/types/proto/v1/externalvs_service";

const props = defineProps<{
  customClass?: string;
  type: ExternalVersionControl_Type;
}>();

const icon = computed(() => {
  let iconType = "";
  switch (props.type) {
    case ExternalVersionControl_Type.AZURE_DEVOPS:
      iconType = "azure-devops";
      break;
    case ExternalVersionControl_Type.GITLAB:
      iconType = "gitlab";
      break;
    case ExternalVersionControl_Type.GITHUB:
      iconType = "github";
      break;
    case ExternalVersionControl_Type.BITBUCKET:
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
