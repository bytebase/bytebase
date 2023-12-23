<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="bg-white px-4 sm:px-6 md:grid md:place-items-center lg:px-8">
    <div class="max-w-max mx-auto">
      <main class="sm:flex">
        <div class="sm:ml-6">
          <div
            class="mt-6 flex space-x-3 sm:border-l sm:border-transparent sm:pl-6"
          >
            <NButton type="primary" @click.prevent="goHome">
              {{ $t("error-page.go-back-home") }}
            </NButton>
          </div>
        </div>
      </main>
    </div>
    <img class="overflow-hidden" src="../assets/illustration/403.webp" alt="" />
  </div>
</template>

<script lang="ts">
import { useRouter } from "vue-router";

export default {
  name: "ThePage403",
  setup() {
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
