<template>
  <BBModal
    :show="authStore.showLoginModal"
    :trap-focus="true"
    :show-close="false"
    :mask-closable="false"
    :header-class="'!hidden'"
  >
    <div style="width: 30vw; height: 70vh" class="flex items-center">
      <div class="flex flex-col justify-center items-center flex-1 space-y-2">
        <Signin :allow-signup="false">
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
import { watch } from "vue";
import { useRoute } from "vue-router";
import { BBModal } from "@/bbkit";
import { useAuthStore } from "@/store";
import Signin from "@/views/auth/Signin.vue";

const route = useRoute();
const authStore = useAuthStore();

const logout = () => {
  authStore.showLoginModal = false;
  authStore.logout();
};

// Auto-close the modal when the route changed.
watch(
  () => route.name,
  () => {
    if (authStore.showLoginModal) {
      authStore.showLoginModal = false;
    }
  }
);
</script>
