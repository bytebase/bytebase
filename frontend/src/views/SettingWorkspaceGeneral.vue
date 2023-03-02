<template>
  <div class="space-y-4 divide-y divide-block-border">
    <BrandingSetting />
    <SecuritySetting />
    <NetworkSetting />
  </div>
</template>

<script lang="ts" setup>
import {
  BrandingSetting,
  SecuritySetting,
  NetworkSetting,
} from "@/components/GeneralSetting";
</script>
