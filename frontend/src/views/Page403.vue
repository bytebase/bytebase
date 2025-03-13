<template>
  <div class="w-full px-4 grid place-items-center py-24">
    <img
      class="w-16 h-auto opacity-80"
      src="@/assets/logo-icon.svg"
      alt="Bytebase"
    />
    <p class="mt-4 text-balance text-center">
      {{ $t("common.missing-required-permission") }}
    </p>
    <div class="mt-12">
      <NButton size="small" @click.prevent="goHome">
        <template #icon><ChevronLeftIcon /></template>
        {{ $t("error-page.go-back-home") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ChevronLeftIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { useRouter } from "vue-router";
import { WORKSPACE_ROOT_MODULE } from "@/router/dashboard/workspaceRoutes";

const router = useRouter();

const goHome = () => {
  router.push({ name: WORKSPACE_ROOT_MODULE });
};
</script>
