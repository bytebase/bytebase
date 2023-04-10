<template>
  <div class="w-full">
    <BBAttention
      :style="'WARN'"
      :title="$t('two-factor.messages.2fa-required')"
      side="CENTER"
    />
    <div class="w-full p-2 sm:p-8 sm:px-16">
      <TwoFactorSetup :cancel-action="handleSetupCancel" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useRouter } from "vue-router";
import { useAuthStore } from "@/store";
import TwoFactorSetup from "./TwoFactorSetup.vue";

const router = useRouter();
const authStore = useAuthStore();

const handleSetupCancel = () => {
  authStore.logout().then(() => {
    router.push({ name: "auth.signin" });
  });
};
</script>
