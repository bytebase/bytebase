<template>
  <div
    class="mx-auto w-full h-full py-6 flex flex-col justify-center items-center"
  >
    <NCard class="w-80 p-8 py-6 shadow-sm">
      <img
        class="h-12 w-auto mx-auto mb-8"
        src="@/assets/logo-full.svg"
        alt="Bytebase"
      />
      <form
        class="w-full mt-4 h-auto flex flex-col justify-start items-center"
        @submit.prevent="challenge"
      >
        <template v-if="state.selectedMFAType === 'OTP'">
          <SmartphoneIcon
            class="w-8 h-auto opacity-60"
          />
          <p class="my-2 mb-4">{{ $t("multi-factor.auth-code") }}</p>
          <NInputOtp
            v-model:value="state.otpCodes"
            @finish="onOtpCodeFinish"
          />
        </template>
        <template v-else-if="state.selectedMFAType === 'RECOVERY_CODE'">
          <heroicons-outline:key class="w-8 h-auto opacity-60" />
          <p class="my-2 mb-4">{{ $t("multi-factor.recovery-code") }}</p>
          <BBTextField
            v-model:value="state.recoveryCode"
            placeholder="XXXXXXXXXX"
            class="w-full"
          />
        </template>
        <div class="w-full mt-4">
          <NButton class="w-full!" attr-type="submit" type="primary">
            {{ $t("common.verify") }}
          </NButton>
        </div>
        <p class="textinfolabel mt-2">
          {{ challengeDescription }}
        </p>
      </form>
      <hr class="my-3" />
      <div class="text-sm mb-2">
        <p class="">{{ $t("multi-factor.other-methods.self") }}:</p>
        <ul class="list-disc list-inside pl-2 pt-1">
          <li v-if="state.selectedMFAType !== 'OTP'">
            <button class="accent-link" @click="state.selectedMFAType = 'OTP'">
              {{ $t("multi-factor.other-methods.use-auth-app.self") }}
            </button>
          </li>
          <li v-if="state.selectedMFAType !== 'RECOVERY_CODE'">
            <button
              class="accent-link"
              @click="state.selectedMFAType = 'RECOVERY_CODE'"
            >
              {{ $t("multi-factor.other-methods.use-recovery-code.self") }}
            </button>
          </li>
        </ul>
      </div>
    </NCard>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { SmartphoneIcon } from "lucide-vue-next";
import { NButton, NCard, NInputOtp } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { BBTextField } from "@/bbkit";
import { useAuthStore } from "@/store";
import { LoginRequestSchema } from "@/types/proto-es/v1/auth_service_pb";

type MFAType = "OTP" | "RECOVERY_CODE";

interface LocalState {
  selectedMFAType: MFAType;
  otpCodes: string[];
  recoveryCode: string;
}

const { t } = useI18n();
const route = useRoute();
const authStore = useAuthStore();
const state = reactive<LocalState>({
  selectedMFAType: "OTP",
  otpCodes: [],
  recoveryCode: "",
});

const mfaTempToken = computed(() => {
  return route.query.mfaTempToken as string;
});

const challengeDescription = computed(() => {
  if (state.selectedMFAType === "OTP") {
    return t("multi-factor.other-methods.use-auth-app.description");
  } else if (state.selectedMFAType === "RECOVERY_CODE") {
    return t("multi-factor.other-methods.use-recovery-code.description");
  } else {
    return "";
  }
});

const onOtpCodeFinish = async (value: string[]) => {
  state.otpCodes = value;
  await challenge();
};

const challenge = async () => {
  const request = create(LoginRequestSchema, {
    mfaTempToken: mfaTempToken.value,
  });
  if (state.selectedMFAType === "OTP") {
    request.otpCode = state.otpCodes.join("");
  } else if (state.selectedMFAType === "RECOVERY_CODE") {
    request.recoveryCode = state.recoveryCode;
  }
  await authStore.login({
    request,
    redirect: true,
  });
};
</script>
