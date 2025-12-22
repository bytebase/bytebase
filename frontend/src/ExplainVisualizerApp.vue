<template>
  <div id="app" class="d-flex">
    <template v-if="storedQuery">
      <!-- PostgreSQL visualization using pev2 -->
      <pev2
        v-if="isPostgreSQL"
        :plan-source="storedQuery.explain"
        :plan-query="storedQuery.statement"
      />
      <!-- MSSQL visualization using html-query-plan -->
      <div
        v-else-if="isMSSQL"
        ref="mssqlContainer"
        class="mssql-query-plan-container"
      />
      <!-- Spanner visualization -->
      <SpannerQueryPlan
        v-else-if="isSpanner"
        :plan-source="storedQuery.explain"
        :plan-query="storedQuery.statement"
      />
      <!-- Fallback for unsupported engines -->
      <div v-else class="unsupported-engine">
        <h2>Unsupported Database Engine</h2>
        <p>Query plan visualization is not available for this database engine.</p>
      </div>
    </template>
    <template v-else>
      <h1>session expired</h1>
    </template>
  </div>
</template>

<script setup lang="ts">
import { Plan as pev2 } from "pev2";
import "pev2/dist/pev2.css";
import "html-query-plan/css/qp.css";
import { parse } from "qs";
import { computed, onMounted, ref } from "vue";
import { SpannerQueryPlan } from "@/components/SpannerQueryPlan";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { readExplainFromToken } from "@/utils/pev2";

// Declare the global QP object from html-query-plan
declare global {
  interface Window {
    QP: {
      showPlan: (
        container: Element,
        planXml: string,
        options?: Record<string, unknown>
      ) => void;
    };
  }
}

const token = computed(() => {
  const query = location.search.replace(/^\?/, "");
  return (parse(query).token as string) || "";
});

const storedQuery = computed(() => readExplainFromToken(token.value));

const isPostgreSQL = computed(
  () => storedQuery.value?.engine === Engine.POSTGRES
);
const isMSSQL = computed(() => storedQuery.value?.engine === Engine.MSSQL);
const isSpanner = computed(() => storedQuery.value?.engine === Engine.SPANNER);

const mssqlContainer = ref<HTMLDivElement>();
const qpLoaded = ref(false);

// Load html-query-plan script dynamically
const loadQueryPlanScript = () => {
  return new Promise<void>((resolve, reject) => {
    // Check if already loaded
    if (window.QP) {
      qpLoaded.value = true;
      resolve();
      return;
    }

    // Check if script already exists
    if (document.getElementById("html-query-plan-script")) {
      qpLoaded.value = true;
      resolve();
      return;
    }

    const script = document.createElement("script");
    script.id = "html-query-plan-script";
    script.src = "/libs/qp.min.js";
    script.onload = () => {
      qpLoaded.value = true;
      resolve();
    };
    script.onerror = () => {
      reject(new Error("Failed to load html-query-plan script"));
    };
    document.head.appendChild(script);
  });
};

onMounted(async () => {
  if (isMSSQL.value && mssqlContainer.value && storedQuery.value) {
    try {
      await loadQueryPlanScript();
      if (window.QP && mssqlContainer.value) {
        window.QP.showPlan(mssqlContainer.value, storedQuery.value.explain);
      }
    } catch (error) {
      console.error("Failed to load query plan visualizer:", error);
    }
  }
});
</script>

<style>
html,
body,
#app {
  height: 100%;
}

.mssql-query-plan-container {
  width: 100%;
  height: 100%;
  overflow: auto;
  padding: 16px;
}

.unsupported-engine {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  padding: 24px;
  text-align: center;
}

.unsupported-engine h2 {
  margin-bottom: 8px;
  color: #666;
}

.unsupported-engine p {
  color: #999;
}

.qp-root {
  width: 100%;
  height: 100%;
}
</style>
