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
import { AUTH_MFA_MODULE } from "@/router/auth";
import { useAuthStore } from "@/store";

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

const trySignin = async (idpName?: string) => {
  if (state.isLoading) return;
  state.isLoading = true;
  try {
    const mfaTempToken = await authStore.login({
      email: state.email,
      password: state.password,
      web: true,
      idpName: idpName,
    });
    if (mfaTempToken) {
      router.push({
        name: AUTH_MFA_MODULE,
        query: {
          mfaTempToken,
          redirect: route.query.redirect as string,
        },
      });
    } else {
      router.push("/");
    }
  } finally {
    state.isLoading = false;
  }
};
</script>
