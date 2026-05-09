<template>
  <ProvideAIContext>
    <Suspense>
      <AIChatToSQL />
    </Suspense>
  </ProvideAIContext>
</template>

<script lang="ts" setup>
/**
 * Self-contained AI chat panel host for use inside a `VueMount` bridge.
 * The bridge spawns an isolated Vue 3 app whose component tree can't see
 * the outer Vue's provide chain — so we re-establish `ProvideAIContext`
 * at the bridge boundary. Pinia + the module-level `aiContextEvents`
 * are singletons, so duplicating the provide doesn't fork state.
 *
 * Mounted from the post-Stage-21 React shells (`StandardPanel`,
 * `Panels`) via `<VueMount component={AIChatToSQLBridgeHost} />`.
 * Removed alongside Stage 22 if/when the AI plugin is ported to React.
 */
import AIChatToSQL from "./AIChatToSQL.vue";
import ProvideAIContext from "./ProvideAIContext.vue";
</script>
