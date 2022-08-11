<template>
  <div
    class="dialog-wrapper fixed top-0 left-0 w-full h-full flex flex-col justify-center items-center"
  >
    <div
      class="dialog-container bg-white relative rounded-lg px-2 pb-6 w-72 h-auto flex flex-col justify-start items-center"
    >
      <span
        class="absolute right-3 top-3 p-px rounded cursor-pointer hover:bg-gray-100 hover:shadow"
        @click="handleCloseButtonClick"
      >
        <heroicons-outline:x class="w-5 h-auto" />
      </span>
      <img src="../../../assets/book-demo.webp" class="w-full h-auto" alt="" />
      <template v-if="!submitted">
        <input
          ref="emailInputRef"
          v-model="email"
          type="text"
          class="w-60 rounded shadow-inner border-gray-300 text-center"
          placeholder="Enter your email"
          @keydown="handleInputKeydown"
        />
        <button
          class="w-60 px-4 mt-4 flex flex-row justify-center items-center leading-10 rounded shadow bg-indigo-600 text-white hover:opacity-80"
          @click="handleRequestButtonClick"
        >
          <heroicons-outline:chat class="w-5 h-auto mr-2" /> Request full demo
        </button>
      </template>
      <template v-else>
        <p class="text-indigo-600 text-xl my-2 font-medium">
          <span class="text-2xl mr-2">🙌</span>Congratulations
        </p>
        <p class="text-sm text-gray-600 leading-10">
          We will contact you in 1 business day!
        </p>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";

const emit = defineEmits(["close"]);

const emailInputRef = ref<HTMLInputElement>();
const email = ref("");
const submitted = ref(false);

const handleInputKeydown = (event: KeyboardEvent) => {
  if (event.code === "Enter") {
    handleRequestButtonClick();
  }
};

const handleCloseButtonClick = () => {
  emit("close");
};

const handleRequestButtonClick = async () => {
  if (email.value === "") {
    return;
  }

  const analytics = (window as any).analytics;
  analytics.identify(email.value, {
    integrations: {
      MailChimp: {
        subscriptionStatus: "subscribed",
      },
    },
  });
  submitted.value = true;
};
</script>

<style scoped>
.dialog-wrapper {
  z-index: 100000;
}
.dialog-wrapper > .dialog-container {
  z-index: 100000;
  box-shadow: 0 0 24px 8px rgb(0 0 0 / 20%);
}
</style>
