<template>
  <form class="space-y-6 px-1" @submit.prevent="trySignin()">
    <div>
      <label
        for="email"
        class="block text-sm font-medium leading-5 text-control"
      >
        {{ $t("common.email") }}
        <span class="text-red-600">*</span>
      </label>
      <div class="mt-1 rounded-md shadow-sm">
        <BBTextField
          v-model:value="state.email"
          required
          :input-props="{
            id: 'email',
            autocomplete: 'on',
            type: 'email',
          }"
          placeholder="jim@example.com"
        />
      </div>
    </div>

    <div>
      <label
        for="password"
        class="flex justify-between text-sm font-medium leading-5 text-control"
      >
        <div>
          {{ $t("common.password") }}
          <span class="text-red-600">*</span>
        </div>
        <router-link
          v-if="props.showForgotPassword"
          :to="{
            path: '/auth/password-forgot',
            query: {
              hint: route.query.hint,
            },
          }"
          class="text-sm font-normal text-control-light hover:underline focus:outline-none"
          tabindex="-1"
        >
          {{ $t("auth.sign-in.forget-password") }}
        </router-link>
      </label>
      <div
        class="relative flex flex-row items-center mt-1 rounded-md shadow-sm"
      >
        <BBTextField
          v-model:value="state.password"
          :type="state.showPassword ? 'text' : 'password'"
          :input-props="{ id: 'password', autocomplete: 'on' }"
          required
        />
        <div
          class="hover:cursor-pointer absolute right-3"
          @click="
            () => {
              state.showPassword = !state.showPassword;
            }
          "
        >
          <EyeIcon v-if="state.showPassword" class="w-4 h-4" />
          <EyeOffIcon v-else class="w-4 h-4" />
        </div>
      </div>
    </div>

    <div class="w-full">
      <NButton
        attr-type="submit"
        type="primary"
        :disabled="!allowSignin"
        :loading="state.isLoading"
        size="large"
        style="width: 100%"
      >
        {{ $t("common.sign-in") }}
      </NButton>
    </div>
  </form>
</template>

<script setup lang="ts">
import { EyeIcon, EyeOffIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBTextField } from "@/bbkit";
import { AUTH_MFA_MODULE, AUTH_PASSWORD_RESET_MODULE } from "@/router/auth";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import { useAppFeature, useAuthStore, useSettingV1Store } from "@/store";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";

interface LocalState {
  email: string;
  password: string;
  showPassword: boolean;
  isLoading: boolean;
}

const props = withDefaults(
  defineProps<{
    email?: string;
    password?: string;
    showPassword?: boolean;
    showForgotPassword?: boolean;
  }>(),
  {
    email: "",
    password: "",
    showPassword: false,
    showForgotPassword: true,
  }
);

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

const state = reactive<LocalState>({
  email: props.email,
  password: props.password,
  showPassword: props.showPassword,
  isLoading: false,
});

const allowSignin = computed(() => {
  return state.email && state.password;
});

const trySignin = async () => {
  if (state.isLoading) return;
  state.isLoading = true;
  try {
    const { mfaTempToken, requireResetPassword } = await authStore.login({
      email: state.email,
      password: state.password,
      web: true,
    });
    if (mfaTempToken) {
      router.push({
        name: AUTH_MFA_MODULE,
        query: {
          mfaTempToken,
          redirect: route.query.redirect as string,
        },
      });
      return;
    }
    if (requireResetPassword) {
      router.push({
        name: AUTH_PASSWORD_RESET_MODULE,
      });
      return;
    }
    try {
      await useSettingV1Store().fetchSettingByName(
        "bb.workspace.profile",
        /* silent */ true
      );
      const mode = useAppFeature("bb.feature.database-change-mode");
      if (mode.value === DatabaseChangeMode.EDITOR) {
        router.replace({
          name: SQL_EDITOR_HOME_MODULE,
        });
      } else {
        router.replace("/");
      }
    } catch {
      router.replace("/");
    }
  } finally {
    state.isLoading = false;
  }
};
</script>
