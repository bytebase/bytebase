<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <img
        class="h-12 w-auto"
        src="../../assets/logo-full.svg"
        alt="Bytebase"
      />
      <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
        {{ $t("auth.sign-in.title") }}
      </h2>
    </div>

    <div class="mt-8">
      <div class="mt-6">
        <form class="space-y-6" @submit.prevent="trySignin">
          <div>
            <label
              for="email"
              class="block text-sm font-medium leading-5 text-control"
            >
              {{ $t("common.email") }}<span class="text-red-600">*</span>
            </label>
            <div class="mt-1 rounded-md shadow-sm">
              <input
                id="email"
                v-model="state.email"
                type="email"
                required
                placeholder="jim@example.com"
                class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-control-placeholder focus:outline-none focus:shadow-outline-blue focus:border-control-border sm:text-sm sm:leading-5"
              />
            </div>
          </div>

          <div>
            <label
              for="password"
              class="flex justify-between text-sm font-medium leading-5 text-control"
            >
              <div>
                {{ $t("common.password")
                }}<span class="text-red-600">*</span>
              </div>
              <router-link
                to="/auth/password-forgot"
                class="text-sm font-normal text-control-light hover:underline focus:outline-none"
              >
                {{ $t("auth.sign-in.forget-password") }}
              </router-link>
            </label>
            <div class="mt-1 rounded-md shadow-sm">
              <input
                id="password"
                v-model="state.password"
                type="password"
                autocomplete="on"
                required
                class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-control-placeholder focus:outline-none focus:shadow-outline-blue focus:border-control-border sm:text-sm sm:leading-5"
              />
            </div>
          </div>

          <div>
            <span class="block w-full rounded-md shadow-sm">
              <button
                type="submit"
                :disabled="!allowSignin"
                class="btn-primary w-full flex justify-center py-2 px-4"
              >
                {{ $t("common.sign-in") }}
              </button>
            </span>
          </div>
        </form>
      </div>
    </div>

    <div class="mt-6 relative">
      <div class="absolute inset-0 flex items-center" aria-hidden="true">
        <div class="w-full border-t border-control-border"></div>
      </div>
      <div class="relative flex justify-center text-sm">
        <span class="pl-2 bg-white text-control">
          {{ $t("auth.sign-in.new-user") }}
        </span>
        <router-link to="/auth/signup" class="accent-link bg-white px-2">
          {{ $t("common.sign-up") }}
        </router-link>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { LoginInfo } from "../../types";
import { isDev, isValidEmail } from "../../utils";

interface LocalState {
  email: string;
  password: string;
}

export default {
  name: "Signin",
  setup() {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      email: "",
      password: "",
    });

    onMounted(() => {
      const demo = store.getters["actuator/isDemo"]();
      state.email = isDev() || demo ? "demo@example.com" : "";
      state.password = isDev() || demo ? "1024" : "";
      // Navigate to signup if needs admin setup.
      // Unable to achieve it in router.beforeEach because actuator/info is fetched async and returns
      // after router has already made the decision on first page load.
      if (store.getters["actuator/needAdminSetup"]()) {
        router.push({ name: "auth.signup", replace: true });
      }
    });

    const allowSignin = computed(() => {
      return isValidEmail(state.email) && state.password;
    });

    const trySignin = () => {
      const loginInfo: LoginInfo = {
        email: state.email,
        password: state.password,
      };
      store.dispatch("auth/login", loginInfo).then(() => {
        router.push("/");
      });
    };

    return {
      state,
      allowSignin,
      trySignin,
    };
  },
};
</script>
