<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <ProvideContext>
        <router-view />
      </ProvideContext>
    </template>
    <template #fallback>
      <span>Loading...</span>
    </template>
  </Suspense>
</template>

<script lang="ts">
import { onErrorCaptured } from "vue";
import { useStore } from "vuex";
import ProvideContext from "./components/ProvideContext.vue";

export default {
  name: "App",
  components: { ProvideContext },
  setup(props, ctx) {
    const store = useStore();

    // Restore previously logged in user if applicable.
    store.dispatch("auth/restoreUser").catch((error: Error) => {
      console.log(error);
    });

    onErrorCaptured((e: any, _, info) => {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: `Internal error occured in '${info}'.`,
        description: e.stack,
      });
      return true;
    });
  },
};
</script>
