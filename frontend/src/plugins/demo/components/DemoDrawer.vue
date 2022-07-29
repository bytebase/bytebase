<template>
  <Transition name="slide">
    <div
      class="demo-drawer-container px-6 py-6 fixed right-0 top-0 w-96 h-screen flex flex-col justify-start items-start shadow-xl bg-white"
    >
      <div
        class="absolute top-4 right-4 w-6 h-6 flex flex-row justify-center items-center rounded cursor-pointer hover:bg-gray-100"
        @click="handleCloseButtonClick"
      >
        <span class="text-2xl -mt-1">&times;</span>
      </div>
      <p class="text-4xl my-2">🎉</p>
      <p class="text-xl font-medium">Demo Completed</p>
      <div class="text-base leading-6 mt-4">
        <p>If you have a question about site</p>
        <p>If you have a question about site</p>
        <p>If you have a question about site</p>
      </div>
      <div class="w-full text-base leading-6 mt-4">
        <p>And here are the other demos:</p>
        <ul class="list-disc ml-6 leading-7 my-1">
          <li>demo1</li>
          <li>demo2</li>
          <li>demo3</li>
          <li>demo4</li>
        </ul>
      </div>
    </div>
  </Transition>
</template>

<script lang="ts" setup>
const emits = defineEmits(["close"]);

const handleCloseButtonClick = () => {
  emits("close");
};
</script>

<style scoped>
.demo-drawer-container {
  z-index: 10001;
}

.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
}
</style>
