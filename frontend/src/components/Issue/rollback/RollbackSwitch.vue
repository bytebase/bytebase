<template>
  <div class="flex items-center h-[30px]">
    <BBSwitch
      :disabled="!allowRollback"
      :value="rollbackEnabled"
      :text="true"
      @toggle="toggleRollback"
    />
  </div>
</template>

<script lang="ts" setup>
import { BBSwitch } from "@/bbkit";
import { useRollbackLogic } from "./common";

const { allowRollback, rollbackEnabled, toggleRollback } = useRollbackLogic();
</script>
