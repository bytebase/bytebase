<template>
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
                @input="onTextEmail"
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

          <div>
            <label
              for="email"
              class="block text-sm font-medium leading-5 text-control"
            >
              Name
            </label>
            <div class="mt-1 rounded-md shadow-sm">
              <input
                id="name"
                type="text"
                v-model="state.name"
                placeholder="Jim Gray"
                class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-control-placeholder focus:outline-none focus:shadow-outline-blue focus:border-control-border sm:text-sm sm:leading-5"
                @input="onTextName"
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
        <router-link to="/auth/signin" class="accent-link px-2 bg-white">
          Sign in
        </router-link>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { SignupInfo } from "../../types";

interface LocalState {
  email: string;
  password: string;
  name: string;
  rememberMe: boolean;
  nameManuallyEdited: boolean;
}

export default {
  name: "Signup",
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      email: "",
      password: "",
      name: "",
      rememberMe: true,
      nameManuallyEdited: false,
    });

    const onTextEmail = () => {
      const email = state.email.trim();
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

    const trySignup = () => {
      const signupInfo: SignupInfo = {
        email: state.email,
        password: state.password,
        name: state.name,
      };
      store
        .dispatch("auth/signup", signupInfo)
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
      onTextEmail,
      onTextName,
      trySignup,
    };
  },
};
</script>
