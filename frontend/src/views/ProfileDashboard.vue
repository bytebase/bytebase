<template>
  <div class="h-screen flex overflow-hidden bg-white">
    <div class="flex flex-col min-w-0 flex-1 overflow-hidden">
      <div class="flex-1 relative z-0 flex overflow-hidden">
        <main
          class="flex-1 relative z-0 overflow-y-auto focus:outline-none xl:order-last"
          tabindex="0"
        >
          <article>
            <!-- Profile header -->
            <div>
              <div class="h-32 w-full bg-accent lg:h-48"></div>
              <div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
                <div class="-mt-20 sm:flex sm:items-end sm:space-x-5">
                  <BBAvatar :size="'huge'" :username="principal.name" />
                  <div
                    class="mt-6 sm:flex-1 sm:min-w-0 sm:flex sm:items-center sm:justify-end sm:space-x-6 sm:pb-1"
                  >
                    <div
                      class="mt-6 flex flex-col justify-stretch space-y-3 sm:flex-row sm:space-y-0 sm:space-x-4"
                    >
                      <button v-if="false" type="button" class="btn-normal">
                        <!-- Heroicon name: solid/mail -->
                        <svg
                          class="-ml-1 mr-2 h-5 w-5 text-control-light"
                          xmlns="http://www.w3.org/2000/svg"
                          viewBox="0 0 20 20"
                          fill="currentColor"
                          aria-hidden="true"
                        >
                          <path
                            d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z"
                          />
                          <path
                            d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z"
                          />
                        </svg>
                        <span>Message</span>
                      </button>
                      <button
                        v-if="isCurrentUser"
                        type="button"
                        class="btn-normal"
                        @click.prevent="editUser"
                      >
                        <!-- Heroicon name: solid/pencil -->
                        <svg
                          class="-ml-1 mr-2 h-5 w-5 text-control-light"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                          xmlns="http://www.w3.org/2000/svg"
                        >
                          <path
                            d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
                          ></path>
                        </svg>
                        <span>Edit</span>
                      </button>
                    </div>
                  </div>
                </div>
                <div class="block mt-6 min-w-0 flex-1">
                  <h1 class="text-2xl font-bold text-main truncate">
                    {{ principal.name }}
                  </h1>
                </div>
              </div>
            </div>

            <!-- Description list -->
            <div class="mt-6 max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
              <dl class="grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-2">
                <div class="sm:col-span-1">
                  <dt class="text-sm font-medium text-control-light">Email</dt>
                  <dd class="mt-1 text-sm text-main">
                    {{ principal.email }}
                  </dd>
                </div>
              </dl>
            </div>
          </article>
        </main>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { PrincipalId } from "../types";

export default {
  name: "ProfileDashboard",
  props: {
    principalId: {
      required: true,
      type: String,
    },
  },
  components: {},
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const principal = computed(() => {
      return store.getters["principal/principalById"](props.principalId);
    });

    const isCurrentUser = computed(() => {
      return currentUser.value.id == principal.value.id;
    });

    const editUser = () => {
      router.push({ name: "setting.profile" });
    };

    return { isCurrentUser, principal, editUser };
  },
};
</script>
