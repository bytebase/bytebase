<template>
  <div class="mx-auto w-full max-w-sm">
    <BytebaseLogo class="mx-auto" />

    <div class="mt-8">
      <NCard>
        <p class="text-xl pl-1 font-medium mb-4">
          {{ $t("common.sign-in-as-admin") }}
        </p>
        <PasswordSigninForm
          :show-forgot-password="false"
          :loading="isLoading"
          @signin="trySignin"
        />
      </NCard>
    </div>
  </div>
  <AuthFooter />
</template>

<script lang="ts" setup>
import { NCard } from "naive-ui";
import { ref } from "vue";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import PasswordSigninForm from "@/components/PasswordSigninForm.vue";
import { useAuthStore } from "@/store";
import { type LoginRequest } from "@/types/proto-es/v1/auth_service_pb";
import AuthFooter from "./AuthFooter.vue";

const isLoading = ref(false);
const authStore = useAuthStore();

const trySignin = async (request: LoginRequest) => {
  if (isLoading.value) return;
  isLoading.value = true;
  try {
    await authStore.login({
      request,
      redirect: true,
    });
  } finally {
    isLoading.value = false;
  }
};
</script>