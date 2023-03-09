<template>
  <BBModal :title="modalTitle" :show-close="false">
    <div class="w-72">
      <div class="w-full mt-4 h-auto flex flex-col justify-start items-center">
        <template v-if="state.selectedMFAType === 'OTP'">
          <heroicons-outline:device-phone-mobile class="w-8 h-auto" />
          <p class="my-2 mb-4">{{ $t("multi-factor.auth-code") }}</p>
          <input
            v-model="state.otpCode"
            placeholder="XXXXXX"
            class="textfield w-full"
            type="text"
          />
        </template>
        <template v-else-if="state.selectedMFAType === 'RECOVERY_CODE'">
          <heroicons-outline:key class="w-8 h-auto" />
          <p class="my-2 mb-4">{{ $t("multi-factor.recovery-code") }}</p>
          <input
            v-model="state.recoveryCode"
            placeholder="XXXXXXXXXX"
            class="textfield w-full"
            type="text"
          />
        </template>
        <button class="btn-success w-full mt-4" @click="challenge">
          <span class="w-full text-center">{{ $t("common.verify") }}</span>
        </button>
        <p class="textinfolabel mt-2">
          {{ challengeDescription }}
        </p>
      </div>
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
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";

type MFAType = "OTP" | "RECOVERY_CODE";

interface LocalState {
  selectedMFAType: MFAType;
  otpCode: string;
  recoveryCode: string;
}

const props = defineProps({
  title: {
    type: String,
    default: "",
  },
  challengeCallback: {
    type: Function,
    required: true,
  },
});

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedMFAType: "OTP",
  otpCode: "",
  recoveryCode: "",
});

const modalTitle = computed(() => {
  return props.title ? props.title : t("multi-factor.self");
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

const challenge = async () => {
  if (state.selectedMFAType === "OTP") {
    await props.challengeCallback({
      otpCode: state.otpCode,
    });
  } else if (state.selectedMFAType === "RECOVERY_CODE") {
    await props.challengeCallback({
      recoveryCode: state.recoveryCode,
    });
  }
};
</script>
