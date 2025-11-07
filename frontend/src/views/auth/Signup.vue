<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <BytebaseLogo class="mx-auto mb-8" />

      <h2 class="text-2xl leading-9 font-medium text-main">
        <template v-if="needAdminSetup">
          <i18n-t
            keypath="auth.sign-up.admin-title"
            tag="p"
            class="text-accent font-semnibold text-center"
          >
            <template #account>
              <span class="text-accent font-semnibold">{{
                $t("auth.sign-up.admin-account")
              }}</span>
            </template>
          </i18n-t>
        </template>
        <template v-else> {{ $t("auth.sign-up.title") }}</template>
      </h2>
    </div>

    <div class="mt-8">
      <div class="mt-6">
        <form class="flex flex-col gap-y-6" @submit.prevent="trySignup">
          <div>
            <label
              for="email"
              class="block text-sm font-medium leading-5 text-control"
            >
              {{ $t("common.email") }}
              <RequiredStar />
            </label>
            <div class="mt-1 rounded-md shadow-xs">
              <BBTextField
                v-model:value="state.email"
                required
                placeholder="jim@example.com"
                :input-props="{ id: 'email' }"
                @input="onTextEmail"
              />
            </div>
          </div>

          <UserPassword
            ref="userPasswordRef"
            v-model:password="state.password"
            v-model:password-confirm="state.passwordConfirm"
            :show-learn-more="false"
            :password-restriction="passwordRestriction"
          />

          <div>
            <label
              for="email"
              class="block text-sm font-medium leading-5 text-control"
            >
              {{ $t("common.username") }}
              <RequiredStar />
            </label>
            <div class="mt-1 rounded-md shadow-xs">
              <BBTextField
                id="name"
                v-model:value="state.name"
                required
                placeholder="Jim Gray"
                @input="onTextName"
              />
            </div>
          </div>

          <div
            v-if="needAdminSetup"
            class="w-full flex flex-row justify-start items-start"
          >
            <NCheckbox v-model:checked="state.acceptTermsAndPolicy">
              <i18n-t
                tag="span"
                keypath="auth.sign-up.accept-terms-and-policy"
                class="select-none"
              >
                <template #terms>
                  <a
                    href="https://www.bytebase.com/terms?source=console"
                    class="text-accent"
                    >{{ $t("auth.sign-up.terms-of-service") }}</a
                  >
                </template>
                <template #policy>
                  <a
                    href="https://www.bytebase.com/privacy?source=console"
                    class="text-accent"
                    >{{ $t("auth.sign-up.privacy-policy") }}</a
                  >
                </template>
              </i18n-t>
            </NCheckbox>
          </div>

          <div class="w-full">
            <NButton
              attr-type="submit"
              type="primary"
              size="large"
              :disabled="!allowSignup"
              :loading="state.isLoading"
              style="width: 100%"
            >
              {{
                needAdminSetup
                  ? $t("auth.sign-up.create-admin-account")
                  : $t("common.sign-up")
              }}
            </NButton>
          </div>
        </form>
      </div>
    </div>

    <div v-if="!needAdminSetup" class="mt-6 relative">
      <div class="absolute inset-0 flex items-center" aria-hidden="true">
        <div class="w-full border-t border-control-border"></div>
      </div>
      <div class="relative flex justify-center text-sm">
        <span class="pl-2 bg-white text-control">
          {{ $t("auth.sign-up.existing-user") }}
        </span>
        <router-link
          :to="{ name: AUTH_SIGNIN_MODULE, query: $route.query }"
          class="accent-link px-2 bg-white"
        >
          {{ $t("common.sign-in") }}
        </router-link>
      </div>
    </div>
  </div>

  <AuthFooter />
</template>

<script lang="ts" setup>
import { NButton, NCheckbox } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, onMounted, reactive, ref } from "vue";
import { BBTextField } from "@/bbkit";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import UserPassword from "@/components/User/Settings/UserPassword.vue";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useActuatorV1Store, useAuthStore } from "@/store";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { isValidEmail } from "@/utils";
import AuthFooter from "./AuthFooter.vue";

interface LocalState {
  email: string;
  password: string;
  passwordConfirm: string;
  name: string;
  nameManuallyEdited: boolean;
  acceptTermsAndPolicy: boolean;
  isLoading: boolean;
}

const actuatorStore = useActuatorV1Store();
const userPasswordRef = ref<InstanceType<typeof UserPassword>>();

const state = reactive<LocalState>({
  email: "",
  password: "",
  passwordConfirm: "",
  name: "",
  nameManuallyEdited: false,
  acceptTermsAndPolicy: true,
  isLoading: false,
});

const { needAdminSetup, disallowSignup, passwordRestriction } =
  storeToRefs(actuatorStore);

const allowSignup = computed(() => {
  return (
    isValidEmail(state.email) &&
    state.password &&
    state.name &&
    !userPasswordRef.value?.passwordHint &&
    !userPasswordRef.value?.passwordMismatch &&
    state.acceptTermsAndPolicy &&
    !disallowSignup.value
  );
});

onMounted(() => {
  if (needAdminSetup.value) {
    state.acceptTermsAndPolicy = false;
  }
});

const onTextEmail = () => {
  const email = state.email.trim().toLowerCase();
  state.email = email;
  if (!state.nameManuallyEdited) {
    const emailParts = email.split("@");
    if (emailParts.length > 0) {
      if (emailParts[0].length > 0) {
        const name = emailParts[0].replace("_", ".");
        const nameParts = name.split(".");
        if (nameParts.length >= 2) {
          state.name = [
            nameParts[0].charAt(0).toUpperCase() + nameParts[0].slice(1),
            nameParts[1].charAt(0).toUpperCase() + nameParts[1].slice(1),
          ].join(" ");
        } else {
          state.name = name.charAt(0).toUpperCase() + name.slice(1);
        }
      }
    }
  }
};

const onTextName = () => {
  const name = state.name.trim();
  state.nameManuallyEdited = name.length > 0;
};

const trySignup = async () => {
  if (state.isLoading) return;
  state.isLoading = true;

  try {
    const signupInfo: Partial<User> = {
      email: state.email,
      password: state.password,
      name: state.name,
    };
    await useAuthStore().signup(signupInfo);
  } finally {
    state.isLoading = false;
  }
};
</script>
