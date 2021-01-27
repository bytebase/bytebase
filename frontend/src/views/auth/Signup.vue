<template>
  <!--
  Tailwind UI components require Tailwind CSS v1.8 and the @tailwindcss/ui plugin.
  Read the documentation to get started: https://tailwindui.com/documentation
-->
  <div class="bg-normal min-h-screen flex">
    <div
      class="flex-1 flex flex-col justify-center py-12 px-4 sm:px-6 lg:flex-none lg:px-20 xl:px-24"
    >
      <div class="mx-auto w-full max-w-sm lg:w-96">
        <div>
          <img class="h-12 w-auto" src="../../assets/logo.svg" alt="Bytebase" />
          <h2 class="mt-6 text-3xl leading-9 font-extrabold text-gray-900">
            Register your account
          </h2>
        </div>

        <div class="mt-8">
          <div class="mt-6">
            <form @submit.prevent="trySignup" class="space-y-6">
              <div>
                <label
                  for="email"
                  class="block text-sm font-medium leading-5 text-gray-700"
                >
                  Email
                </label>
                <div class="mt-1 rounded-md shadow-sm">
                  <input
                    id="email"
                    type="email"
                    v-model="state.email"
                    required
                    class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-gray-400 focus:outline-none focus:shadow-outline-blue focus:border-blue-300 sm:text-sm sm:leading-5"
                  />
                </div>
              </div>

              <div>
                <label
                  for="password"
                  class="block text-sm font-medium leading-5 text-gray-700"
                >
                  Password
                </label>
                <div class="mt-1 rounded-md shadow-sm">
                  <input
                    id="password"
                    type="password"
                    v-model="state.password"
                    required
                    class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-gray-400 focus:outline-none focus:shadow-outline-blue focus:border-blue-300 sm:text-sm sm:leading-5"
                  />
                </div>
              </div>

              <div class="flex items-center justify-between">
                <div class="flex items-center">
                  <input
                    id="remember_me"
                    type="checkbox"
                    v-model="state.rememberMe"
                    class="text-accent form-checkbox h-4 w-4"
                  />
                  <label
                    for="remember_me"
                    class="ml-2 block text-sm leading-5 text-gray-900"
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
            <span class="pl-2 bg-normal text-gray-500">
              Already have an account?
            </span>
            <router-link
              to="signin"
              class="text-accent bg-normal px-2 font-medium hover:underline focus:outline-none focus:underline"
            >
              Sign in
            </router-link>
          </div>
        </div>
      </div>
    </div>
    <div class="hidden lg:block relative w-0 flex-1">
      <img
        class="absolute inset-0 h-full w-full object-cover"
        src="../../assets/Signin-splash.jpeg"
        alt=""
      />
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
        type: "signupInfo",
        attributes: {
          username: state.email,
          password: state.password,
        },
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
