<template>
  <div class="relative">
    <button
      type="button"
      @click.prevent="$refs.menu.toggle($event)"
      @contextmenu.capture.prevent="$refs.menu.toggle($event)"
      class="flex text-sm text-white focus:outline-none focus:shadow-solid"
      id="user-menu"
      aria-label="User menu"
      aria-haspopup="true"
    >
      <BBAvatar :username="currentUser.attributes.name"> </BBAvatar>
    </button>
    <BBContextMenu
      ref="menu"
      class="z-10 origin-top-left absolute right-0 mt-2 w-32 rounded-md shadow-lg"
    >
      <div v-if="isDevOrDemo">
        <div
          v-if="currentUser.attributes.email != 'jerry@example.com'"
          class="py-1"
        >
          <div
            @click.prevent="switchToDBA"
            class="block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
            role="menuitem"
          >
            Switch to DBA
          </div>
        </div>
        <div
          v-if="currentUser.attributes.email != 'tom@example.com'"
          class="py-1"
        >
          <div
            @click.prevent="switchToDeveloper"
            class="block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
            role="menuitem"
          >
            Switch to Dev
          </div>
        </div>
        <div class="border-t border-gray-100"></div>
      </div>
      <div class="py-1">
        <router-link
          to="/setting"
          class="block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
          role="menuitem"
          >Settings</router-link
        >
      </div>
      <div class="border-t border-gray-100"></div>
      <div class="py-1">
        <a
          href="#"
          class="block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
          role="menuitem"
          >Changelog</a
        >
        <a
          href="#"
          class="block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
          role="menuitem"
          >Support</a
        >
      </div>
      <div class="border-t border-gray-100"></div>
      <div class="py-1">
        <a
          @click.prevent="logout"
          class="block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
          role="menuitem"
          >Logout</a
        >
      </div>
    </BBContextMenu>
  </div>
</template>

<script lang="ts">
import { inject } from "vue";
import { useStore } from "vuex";
import { UserStateSymbol } from "../components/ProvideUser.vue";
import { User } from "../types";

export default {
  name: "ProfileDropdown",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser = inject<User>(UserStateSymbol);

    const logout = () => {
      store
        .dispatch("auth/logout")
        .then(() => {
          // Just do a reload. router.push won't refresh the app.vue page.
          // It's acceptable to use reload for logout.
          location.reload();
        })
        .catch((error: Error) => {
          console.log(error);
          return;
        });
    };

    const switchToDBA = () => {
      store
        .dispatch("auth/login", {
          type: "loginInfo",
          attributes: {
            username: "jerry@example.com",
            password: "aaa",
          },
        })
        .then(() => {
          // Do a full page reload to avoid stale UI state.
          location.replace("/");
        });
    };

    const switchToDeveloper = () => {
      store
        .dispatch("auth/login", {
          type: "loginInfo",
          attributes: {
            username: "tom@example.com",
            password: "aaa",
          },
        })
        .then(() => {
          // Do a full page reload to avoid stale UI state.
          location.replace("/");
        });
    };

    return {
      currentUser,
      logout,
      switchToDBA,
      switchToDeveloper,
    };
  },
};
</script>
