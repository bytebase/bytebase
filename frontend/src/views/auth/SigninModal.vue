<template>
  <BBModal
    :show="true"
    :trap-focus="true"
    :show-close="false"
    :mask-closable="false"
    :header-class="'hidden!'"
  >
    <div class="flex items-center w-auto md:min-w-96 max-w-full h-auto md:py-4">
      <div class="flex flex-col justify-center items-center flex-1 gap-y-2">
        <Signin
          :redirect="false"
          :redirect-url="currentPath"
          :allow-signup="false"
        >
          <template #footer>
            <NButton quaternary size="small" @click="logout">
              {{ $t("common.logout") }}
            </NButton>
          </template>
        </Signin>
      </div>
    </div>
  </BBModal>
</template>

<script lang="tsx" setup>
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useRoute } from "vue-router";
import { BBModal } from "@/bbkit";
import { useAuthStore } from "@/store";
import Signin from "@/views/auth/Signin.vue";

const authStore = useAuthStore();
const route = useRoute();

const logout = () => {
  authStore.logout();
};

const currentPath = computed(() => route.fullPath);
</script>
