<template>
  <div class="min-h-screen overflow-hidden flex">
    <div
      v-if="showBrandingImage"
      class="hidden bg-white lg:block relative w-0 flex-1"
    >
      <img
        v-if="route === AUTH_SIGNUP_MODULE"
        class="absolute inset-0 h-full w-full object-cover"
        src="@/assets/illustration/signup.webp"
        alt=""
      />
      <img
        v-else
        class="absolute inset-0 h-full w-full object-cover"
        src="@/assets/illustration/signin.webp"
        alt=""
      />
    </div>
    <div
      class="relative mx-auto flex-1 flex flex-col justify-center py-12 pb-24 px-4 sm:px-6 lg:flex-none lg:px-20 lg:w-1/2 xl:px-24"
    >
      <router-view />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { AUTH_SIGNUP_MODULE } from "@/router/auth";
import { useSubscriptionV1Store } from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";

const router = useRouter();
const subscriptionStore = useSubscriptionV1Store();

const route = computed(() => {
  return router.currentRoute.value.name;
});

const showBrandingImage = computed(() => {
  return (
    subscriptionStore.currentPlan !== PlanType.ENTERPRISE ||
    subscriptionStore.isTrialing
  );
});
</script>
