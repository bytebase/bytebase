<template>
  <div class="w-full">
    <BBAttention type="warning">
      {{ $t("two-factor.messages.2fa-required") }}
    </BBAttention>
    <div class="w-full p-2 sm:p-8 sm:px-16">
      <TwoFactorSetup :cancel-action="handleSetupCancel" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { BBAttention } from "@/bbkit";
import { useAuthStore } from "@/store";
import TwoFactorSetup from "./TwoFactorSetup.vue";

const authStore = useAuthStore();

const handleSetupCancel = () => {
  authStore.logout();
};
</script>
