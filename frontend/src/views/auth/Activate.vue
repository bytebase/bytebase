<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <img
        class="h-12 w-auto"
        src="../../assets/logo-full.svg"
        alt="Bytebase"
      />
      <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
        <i18n-t keypath="auth.activate.title" tag="p">
          <template #type>
            <span class="text-accent font-semnibold">{{
              state.role.charAt(0).toUpperCase() +
              state.role.slice(1).toLowerCase()
            }}</span>
          </template>
        </i18n-t>
      </h2>
    </div>

    <div class="mt-8">
      <div class="mt-6">
        <form class="space-y-6" @submit.prevent="tryActivate">
          <div>
            <label
              for="email"
              class="block text-sm font-medium leading-5 text-control"
            >
              {{ $t("common.email") }}
            </label>
            <div class="mt-2 text-base font-medium leading-5 text-accent">
              {{ state.email }}
            </div>
          </div>
          <div>
            <label
              for="password"
              class="block text-sm font-medium leading-5 text-control"
            >
              {{ $t("common.password") }}
              <span class="text-red-600">*</span>
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
            <label
              for="email"
              class="block text-sm font-medium leading-5 text-control"
            >
              {{ $t("common.username") }}
            </label>
            <div class="mt-1 rounded-md shadow-sm">
              <input
                id="name"
                v-model="state.name"
                type="text"
                placeholder="Jim Gray"
                class="appearance-none block w-full px-3 py-2 border border-control-border rounded-md placeholder-control-placeholder focus:outline-none focus:shadow-outline-blue focus:border-control-border sm:text-sm sm:leading-5"
              />
            </div>
          </div>

          <div>
            <span class="block w-full rounded-md shadow-sm">
              <button
                type="submit"
                class="btn-success w-full flex justify-center py-2 px-4"
              >
                {{ $t("common.activate") }}
              </button>
            </span>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, reactive } from "vue";
import { useRouter } from "vue-router";
import { ActivateInfo, RoleType } from "../../types";
import { useAuthStore } from "@/store";

interface LocalState {
  email: string;
  password: string;
  name: string;
  role: RoleType;
}

export default defineComponent({
  name: "Activate",
  setup() {
    const router = useRouter();
    const token = router.currentRoute.value.query.token as string;

    // TODO(tianzhou): Get info from activate token
    const state = reactive<LocalState>({
      email: "bob@example.com",
      password: "",
      name: "Bob Invited",
      role: "DEVELOPER",
    });

    const tryActivate = () => {
      const activateInfo: ActivateInfo = {
        email: state.email,
        password: state.password,
        name: state.name,
        token: token,
      };
      useAuthStore()
        .activate(activateInfo)
        .then(() => {
          router.push("/");
        });
    };

    return {
      state,
      tryActivate,
    };
  },
});
</script>
