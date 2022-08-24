<template>
  <div class="d-flex h-full">
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
import { computed } from "vue";
import { useRoute } from "vue-router";
import { Plan as pev2 } from "pev2";
import "bootstrap/dist/css/bootstrap.css";
import "pev2/dist/style.css";
import { readExplainFromToken } from "@/utils";

const route = useRoute();

const token = computed(() => (route.query.token as string) || "");

const storedQuery = computed(() => readExplainFromToken(token.value));
</script>

<style>
html,
body,
#app,
.n-config-provider {
  height: 100%;
}
</style>
