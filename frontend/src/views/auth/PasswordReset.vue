<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <img
        class="h-12 w-auto"
        src="../../assets/logo-full.svg"
        alt="Bytebase"
      />
      <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
        Reset your password
      </h2>
    </div>

    <div class="mt-8">
      <div class="mt-6">
        <form class="space-y-6" @submit.prevent="tryReset">
          <div>
            <label
              for="email"
              class="block text-base font-medium leading-5 text-control"
            >
              Enter your email address and we will send you a password reset
              link
            </label>
            <div class="mt-4 rounded-md shadow-sm">
              <input
                id="email"
                v-model="state.email"
                type="email"
                required
                placeholder="jim@example.com"
                class="
                  appearance-none
                  block
                  w-full
                  px-3
                  py-2
                  border border-control-border
                  rounded-md
                  placeholder-control-placeholder
                  focus:outline-none
                  focus:shadow-outline-blue
                  focus:border-control-border
                  sm:text-sm sm:leading-5
                "
              />
            </div>
          </div>

          <div>
            <span class="block w-full rounded-md shadow-sm">
              <button
                type="submit"
                :disabled="!allowReset"
                class="btn-primary w-full flex justify-center py-2 px-4"
              >
                Send reset link
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
        <router-link to="/auth/signin" class="accent-link bg-white px-2">
          Return to Sign in
        </router-link>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { isValidEmail } from "../../utils";

interface LocalState {
  email: string;
}

export default {
  name: "PasswordReset",
  setup() {
    const state = reactive<LocalState>({
      email: "",
    });

    const allowReset = computed(() => {
      return isValidEmail(state.email);
    });

    const tryReset = () => {
      console.log("Reset Email", state.email);
    };

    return { state, allowReset, tryReset };
  },
};
</script>
