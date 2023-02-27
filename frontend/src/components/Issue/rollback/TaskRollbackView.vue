<template>
  <div v-if="rollbackUIType !== 'NONE'" class="contents">
    <h2 class="textlabel flex items-center">
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
    </h2>

    <div class="col-span-2 flex items-center">
      <RollbackSwitch v-if="rollbackUIType === 'SWITCH'" />
      <RollbackStatus v-if="rollbackUIType === 'FULL'" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";

import RollbackSwitch from "./RollbackSwitch.vue";
import RollbackStatus from "./RollbackStatus.vue";
import { useRollbackLogic } from "./common";

const { rollbackUIType } = useRollbackLogic();
</script>
