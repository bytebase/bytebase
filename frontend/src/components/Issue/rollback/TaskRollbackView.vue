<template>
  <div v-if="rollbackUIType !== 'NONE'" class="contents">
    <h2 class="textlabel flex items-center">
      <span class="mr-1">{{ $t("task.rollback.sql-rollback") }}</span>
      <!-- TODO(Jim): bring this icon back when user doc is ready -->
      <!-- <heroicons-outline:question-mark-circle class="h-4 w-4" /> -->
    </h2>

    <div class="col-span-2 flex items-center">
      <RollbackSwitch v-if="rollbackUIType === 'SWITCH'" />
      <RollbackStatus v-if="rollbackUIType === 'FULL'" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import RollbackSwitch from "./RollbackSwitch.vue";
import RollbackStatus from "./RollbackStatus.vue";
import { useRollbackLogic } from "./common";

const { rollbackUIType } = useRollbackLogic();
</script>
