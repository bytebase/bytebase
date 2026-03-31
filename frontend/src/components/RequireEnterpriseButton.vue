<template>
  <NButton v-bind="$attrs" @click="requireSubscription">
    <template #icon>
      <slot name="icon"></slot>
    </template>
    <slot name="default"></slot>
  </NButton>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";

const requireSubscription = () => {
  window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
};
</script>
