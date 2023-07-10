<template>
  <div class="flex items-center gap-x-3">
    <h2 class="flex flex-col items-start">
      <div class="flex gap-x-1 items-center textlabel">
        <span class="mr-1">{{ $t("common.when") }}</span>
        <NTooltip>
          <template #trigger>
            <heroicons-outline:question-mark-circle class="h-4 w-4" />
          </template>
          <div class="w-60">
            {{ $t("task.earliest-allowed-time-hint") }}
          </div>
        </NTooltip>
      </div>

      <div class="text-gray-600 text-xs">
        {{ "UTC" + dayjs().format("ZZ") }}
      </div>
    </h2>
    <div class="issue-debug">(put `When` here if needed)</div>
  </div>
</template>

<script setup lang="ts"></script>
