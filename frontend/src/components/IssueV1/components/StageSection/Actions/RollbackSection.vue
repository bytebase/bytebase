<template>
  <div class="flex items-center gap-x-3">
    <div class="textlabel flex items-center">
      <span class="mr-1">{{ $t("task.rollback.sql-rollback") }}</span>
      <NTooltip>
        <template #trigger>
          <heroicons-outline:question-mark-circle class="h-4 w-4" />
        </template>
        <i18n-t
          tag="div"
          keypath="task.rollback.sql-rollback-tips"
          class="whitespace-pre-line"
        >
          <template #link>
            <LearnMoreLink
              url="https://www.bytebase.com/docs/change-database/rollback-data-changes?source=console"
              color="light"
            />
          </template>
        </i18n-t>
      </NTooltip>
    </div>
    <div class="issue-debug">Rollback section if needed</div>
  </div>
</template>

<script setup lang="ts"></script>
