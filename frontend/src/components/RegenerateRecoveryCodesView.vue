<template>
  <RecoveryCodesView
    :recovery-codes="props.recoveryCodes"
    @download="state.recoveryCodesDownloaded = true"
  />
  <div class="w-112 mx-auto flex flex-row justify-between items-center mb-8">
    <button class="btn-cancel" @click="emit('close')">
      {{ $t("common.cancel") }}
    </button>
    <button
      class="btn-primary"
      :disabled="!state.recoveryCodesDownloaded"
      @click="regenerateRecoveryCodes"
    >
      {{ $t("two-factor.setup-steps.recovery-codes-saved") }}
    </button>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useAuthStore, useUserStore } from "@/store";
import { UpdateUserRequest } from "@/types/proto/v1/auth_service";
import RecoveryCodesView from "./RecoveryCodesView.vue";

interface LocalState {
  recoveryCodesDownloaded: boolean;
}

const props = withDefaults(
  defineProps<{
    recoveryCodes: string[];
  }>(),
  {
    recoveryCodes: () => [],
  }
);

const emit = defineEmits(["close"]);

const { t } = useI18n();
const authStore = useAuthStore();
const userStore = useUserStore();
const state = reactive<LocalState>({
  recoveryCodesDownloaded: false,
});

const currentUser = computed(() => authStore.currentUser);

onMounted(() => {
  regenerateTempMfaSecret();
});

const regenerateTempMfaSecret = async () => {
  await userStore.updateUser(
    UpdateUserRequest.fromPartial({
      user: {
        name: currentUser.value.name,
      },
      updateMask: [],
      regenerateTempMfaSecret: true,
    })
  );
  await authStore.refreshUserIfNeeded(currentUser.value.name);
};

const regenerateRecoveryCodes = async () => {
  await userStore.updateUser(
    UpdateUserRequest.fromPartial({
      user: {
        name: currentUser.value.name,
      },
      updateMask: [],
      regenerateRecoveryCodes: true,
    })
  );
  await authStore.refreshUserIfNeeded(currentUser.value.name);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("two-factor.messages.recovery-codes-regenerated"),
  });
  emit("close");
};
</script>
