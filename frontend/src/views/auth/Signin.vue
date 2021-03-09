<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <img class="h-12 w-auto" src="../../assets/logo.svg" alt="Bytebase" />
      <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
        Sign in to your account
      </h2>
    </div>

    <div class="mt-8">
      <div class="mt-6">
        <form @submit.prevent="trySignin" class="space-y-6">
          <div>
            <label
              for="email"
              class="block text-sm font-medium leading-5 text-control"
            >
              Email<span class="text-red-600">*</span>
            </label>
            <div class="mt-1 rounded-md shadow-sm">
              <input
                id="email"
                type="email"
                v-model="state.email"
                required
                placeholder="jim@example.com"
                class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-control-placeholder focus:outline-none focus:shadow-outline-blue focus:border-control-border sm:text-sm sm:leading-5"
              />
            </div>
          </div>

          <div>
            <label
              for="password"
              class="block text-sm font-medium leading-5 text-control"
            >
              Password<span class="text-red-600">*</span>
            </label>
            <div class="mt-1 rounded-md shadow-sm">
              <input
                id="password"
                type="password"
                autocomplete="on"
                v-model="state.password"
                required
                class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-control-placeholder focus:outline-none focus:shadow-outline-blue focus:border-control-border sm:text-sm sm:leading-5"
              />
            </div>
          </div>

          <div class="flex items-center justify-start">
            <div class="text-sm leading-5">
              <router-link to="/auth/password-reset" class="link">
                Forgot your password?
              </router-link>
            </div>
          </div>

          <div>
            <span class="block w-full rounded-md shadow-sm">
              <button
                type="submit"
                class="btn-primary w-full flex justify-center py-2 px-4"
              >
                Sign in
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
        <span class="pl-2 bg-white text-control"> New to Bytebase? </span>
        <router-link to="/auth/signup" class="accent-link bg-white px-2">
          Sign up
        </router-link>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { LoginInfo } from "../../types";

interface LocalState {
  email: string;
  password: string;
}

export default {
  name: "Signin",
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      email: "",
      password: "",
    });

    const trySignin = () => {
      const loginInfo: LoginInfo = {
        email: state.email,
        password: state.password,
      };
      store
        .dispatch("auth/login", loginInfo)
        .then(() => {
          router.push("/");
        })
        .catch((error: Error) => {
          console.log(error);
          return;
        });
    };

    return {
      state,
      trySignin,
    };
  },
};
</script>
