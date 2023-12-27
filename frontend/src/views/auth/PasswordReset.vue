<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <img
        class="h-12 w-auto"
        src="../../assets/logo-full.svg"
        alt="Bytebase"
      />
      <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
        {{ $t("auth.password-reset.title") }}
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
              {{ $t("auth.password-reset.content") }}
            </label>
            <div class="mt-4 rounded-md shadow-sm">
              <BBTextField
                v-model:value="state.email"
                required
                placeholder="jim@example.com"
              />
            </div>
          </div>

          <div class="w-full">
            <NButton
              attr-type="submit"
              type="primary"
              :disabled="!allowReset"
              style="width: 100%"
            >
              {{ $t("auth.password-reset.send-reset-link") }}
            </NButton>
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
          {{ $t("auth.password-reset.return-to-sign-in") }}
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
