<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div
    class="
      bg-white
      px-4
      py-16
      sm:px-6 sm:py-24
      md:grid md:place-items-center
      lg:px-8
    "
  >
    <div class="max-w-max mx-auto">
      <main class="sm:flex">
        <p class="text-4xl font-extrabold text-accent sm:text-5xl">403</p>
        <div class="sm:ml-6">
          <div class="sm:border-l sm:border-block-border sm:pl-6">
            <h1
              class="
                text-4xl
                font-extrabold
                text-main
                tracking-tight
                sm:text-5xl
              "
            >
              Access denied
            </h1>
          </div>
          <div
            class="
              mt-6
              flex
              space-x-3
              sm:border-l sm:border-transparent sm:pl-6
            "
          >
            <button class="btn-primary" @click.prevent="goHome">
              Go back home
            </button>
          </div>
        </div>
      </main>
    </div>
  </div>
</template>

<script lang="ts">
import { useRouter } from "vue-router";
export default {
  name: "403",
  components: {},
  setup(props, ctx) {
    const router = useRouter();

    const goHome = () => {
      router.push({ name: "workspace.home" });
    };

    return {
      goHome,
    };
  },
};
</script>
