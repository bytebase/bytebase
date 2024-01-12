<template>
  <div
    v-if="rollbackUIType !== 'NONE'"
    class="flex items-center justify-between gap-x-2 min-h-[34px] whitespace-nowrap"
  >
    <NTooltip>
      <template #trigger>
        <div class="textlabel flex items-center">
          {{ $t("task.rollback.sql-rollback") }}
        </div>
      </template>
      <template #default>
        <i18n-t
          tag="div"
          keypath="task.rollback.sql-rollback-tips"
          class="whitespace-pre-line"
        >
          <template #link>
            <HideInStandaloneMode>
              <LearnMoreLink
                url="https://www.bytebase.com/docs/change-database/rollback-data-changes?source=console"
                color="light"
              />
            </HideInStandaloneMode>
          </template>
        </i18n-t>
      </template>
    </NTooltip>
    <div class="">
      <RollbackSwitch v-if="rollbackUIType === 'SWITCH'" />
      <RollbackStatus v-if="rollbackUIType === 'FULL'" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { NTooltip } from "naive-ui";
import RollbackStatus from "./RollbackStatus.vue";
import RollbackSwitch from "./RollbackSwitch.vue";
import { useRollbackContext } from "./common";

const rollbackContext = useRollbackContext();

const { rollbackUIType } = rollbackContext;
</script>
