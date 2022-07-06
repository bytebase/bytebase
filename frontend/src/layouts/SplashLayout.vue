<template>
  <div class="min-h-screen overflow-hidden flex">
    <div class="hidden bg-white lg:block relative w-0 flex-1">
      <img
        v-if="route == 'auth.signin'"
        class="absolute inset-0 h-full w-full object-cover"
        src="../assets/illustration/signin.webp"
        alt=""
      />
      <img
        v-else-if="route == 'auth.signup'"
        class="absolute inset-0 h-full w-full object-cover"
        src="../assets/illustration/signup.webp"
        alt=""
      />
    </div>
    <div
      class="relative flex-1 flex flex-col justify-center py-12 pb-24 px-4 sm:px-6 lg:flex-none lg:px-20 lg:w-1/2 xl:px-24"
    >
      <router-view />
    </div>
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";

export default {
  name: "SplashLayout",
  setup() {
    const router = useRouter();

    const route = computed(() => {
      return router.currentRoute.value.name;
    });

    return {
      route,
    };
  },
};
</script>
