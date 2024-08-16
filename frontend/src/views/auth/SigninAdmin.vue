<template>
  <div class="mx-auto w-full max-w-sm">
    <BytebaseLogo class="mx-auto" />

    <div class="mt-8">
      <NCard>
        <p class="text-xl pl-1 font-medium mb-4">
          {{ $t("common.sign-in-as-admin") }}
        </p>
        <PasswordSigninForm :show-forgot-password="false" />
      </NCard>
    </div>
  </div>
  <AuthFooter />
</template>

<script lang="ts" setup>
import { NCard } from "naive-ui";
import { watchEffect } from "vue";
import { useRouter } from "vue-router";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import PasswordSigninForm from "@/components/PasswordSigninForm.vue";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useActuatorV1Store } from "@/store";
import AuthFooter from "./AuthFooter.vue";

const router = useRouter();
const actuatorStore = useActuatorV1Store();

watchEffect(() => {
  if (!actuatorStore.disallowPasswordSignin) {
    router.push({ name: AUTH_SIGNIN_MODULE, replace: true });
  }
});
</script>
