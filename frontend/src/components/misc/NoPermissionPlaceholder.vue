<template>
  <NoDataPlaceholder :src="'@/assets/illustration/403.webp'">
    <template #image>
      <img src="@/assets/illustration/403.webp" class="!w-auto max-h-[20vh]" />
    </template>
    <p class="textlabel">{{ $t("common.missing-permission") }}</p>
  </NoDataPlaceholder>
</template>

<script lang="ts" setup></script>
