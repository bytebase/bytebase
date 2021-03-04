<template>
  <ProvideUser>
    <div>
      <!-- Suspense is experimental, be aware of the potential change -->
      <Suspense>
        <template #default>
          <div>
            <ProvideEnvironment>
              <router-view />
            </ProvideEnvironment>
          </div>
        </template>
        <template #fallback>
          <span>Loading...</span>
        </template>
      </Suspense>
    </div>
  </ProvideUser>
</template>

<script lang="ts">
import ProvideEnvironment from "./components/ProvideEnvironment.vue";
import ProvideUser from "./components/ProvideUser.vue";

export default {
  name: "App",
  components: { ProvideEnvironment, ProvideUser },
  setup(props, ctx) {},
};
</script>
