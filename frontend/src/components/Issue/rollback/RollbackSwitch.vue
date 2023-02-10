<template>
  <BBCheckbox
    :disabled="!allowRollback"
    :value="rollbackEnabled"
    @toggle="toggleRollback"
  />
</template>

<script lang="ts" setup>
import { BBCheckbox } from "@/bbkit";
import { useRollbackLogic } from "./common";

const { allowRollback, rollbackEnabled, toggleRollback } = useRollbackLogic();
</script>
