<template>
  <!--
  Tailwind UI components require Tailwind CSS v1.8 and the @tailwindcss/ui plugin.
  Read the documentation to get started: https://tailwindui.com/documentation
-->
  <div class="min-h-screen flex">
    <div class="hidden bg-main lg:block relative w-0 flex-1">
      <!-- <img
        class="absolute inset-0 h-full w-full object-cover"
        src="../../assets/signin-splash.jpeg"
        alt=""
      /> -->
      <div class="absolute top-0 right-0 p-8">
        <h1 class="text-right text-4xl font-semibold tracking-tight space-y-2">
          <span class="block text-white">Simple systems work</span>
          <span class="block text-white">complex don't</span>
        </h1>
        <p
          class="mt-6 text-right text-xl font-medium tracking-tight text-white"
        >
          Jim Gray
        </p>
      </div>
    </div>
    <div
      class="flex-1 flex flex-col justify-center py-12 px-4 sm:px-6 lg:flex-none lg:px-20 lg:w-1/2 xl:px-24"
    >
      <div class="mx-auto w-full max-w-sm">
        <div>
          <img class="h-12 w-auto" src="../../assets/logo.svg" alt="Bytebase" />
          <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
            Register your account
          </h2>
        </div>

        <div class="mt-8">
          <div class="mt-6">
            <form @submit.prevent="trySignup" class="space-y-6">
              <div>
                <label
                  for="email"
                  class="block text-sm font-medium leading-5 text-control"
                >
                  Email
                </label>
                <div class="mt-1 rounded-md shadow-sm">
                  <input
                    id="email"
                    type="email"
                    v-model="state.email"
                    required
                    class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-control-placeholder focus:outline-none focus:shadow-outline-blue focus:border-control-border sm:text-sm sm:leading-5"
                  />
                </div>
              </div>

              <div>
                <label
                  for="password"
                  class="block text-sm font-medium leading-5 text-control"
                >
                  Password
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

              <div class="flex items-center justify-between">
                <div class="flex items-center">
                  <input
                    id="remember_me"
                    type="checkbox"
                    v-model="state.rememberMe"
                    class="text-control form-checkbox h-4 w-4"
                  />
                  <label
                    for="remember_me"
                    class="ml-2 block text-sm leading-5 text-main"
                  >
                    Remember me
                  </label>
                </div>
              </div>

              <div>
                <span class="block w-full rounded-md shadow-sm">
                  <button
                    type="submit"
                    class="btn-primary w-full flex justify-center py-2 px-4"
                  >
                    Register
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
              Already have an account?
            </span>
            <router-link to="signin" class="accent-link px-2 bg-white">
              Sign in
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "vue";
import { useStore } from "vuex";
import { SignupInfo } from "../../types";

interface LocalState {
  email: string;
  password: string;
  rememberMe: boolean;
}

export default {
  name: "Signup",
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      email: "",
      password: "",
      rememberMe: true,
    });

    const trySignup = () => {
      const signupInfo: SignupInfo = {
        username: state.email,
        password: state.password,
      };
      store
        .dispatch("auth/signup", signupInfo)
        .then(() => {
          // Do a full page reload to avoid stale UI state.
          location.replace("/");
        })
        .catch((error: Error) => {
          console.log(error);
          return;
        });
    };

    return {
      state,
      trySignup,
    };
  },
};
</script>
