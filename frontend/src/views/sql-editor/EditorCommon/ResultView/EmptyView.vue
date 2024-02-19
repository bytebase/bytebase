<template>
  <div
    class="text-md font-normal"
    :class="[dark ? 'text-matrix-green-hover' : 'text-control-light']"
  >
    {{ $t("sql-editor.no-data-available") }}
  </div>
</template>

<script lang="ts" setup>
import { useSQLResultViewContext } from "./context";

const { dark } = useSQLResultViewContext();
</script>
