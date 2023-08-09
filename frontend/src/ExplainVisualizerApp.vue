<template>
  <div id="app" class="d-flex">
    <pev2
      v-if="storedQuery"
      :plan-source="storedQuery.explain"
      :plan-query="storedQuery.statement"
    />
    <template v-else>
      <h1>session expired</h1>
    </template>
  </div>
</template>

<script setup lang="ts">
import { Plan as pev2 } from "pev2";
import "pev2/dist/style.css";
import { parse } from "qs";
import { computed } from "vue";
import { readExplainFromToken } from "@/utils/pev2";

const token = computed(() => {
  const query = location.search.replace(/^\?/, "");
  return (parse(query).token as string) || "";
});

const storedQuery = computed(() => readExplainFromToken(token.value));
</script>

<style>
html,
body,
#app {
  height: 100%;
}
</style>
