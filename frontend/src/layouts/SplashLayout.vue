<template>
  <div class="min-h-screen flex">
    <div class="hidden bg-main lg:block relative w-0 flex-1">
      <!-- <img
        class="absolute inset-0 h-full w-full object-cover"
        src="../../assets/signin-splash.jpeg"
        alt=""
      /> -->
      <div class="absolute top-0 right-0 p-8">
        <h1 class="text-right text-4xl font-semibold tracking-tight space-y-2">
          <span class="block text-white">Simple systems work</span>
          <span class="block text-white">complex don't</span>
        </h1>
        <p
          class="mt-6 text-right text-xl font-medium tracking-tight text-white"
        >
          Jim Gray
        </p>
      </div>
    </div>
    <div
      class="flex-1 flex flex-col justify-center py-12 px-4 sm:px-6 lg:flex-none lg:px-20 lg:w-1/2 xl:px-24"
    >
      <router-view />
    </div>
  </div>
</template>

<script lang="ts">
export default {
  name: "SettingLayout",
  components: {},
  setup(props, ctx) {},
};
</script>
