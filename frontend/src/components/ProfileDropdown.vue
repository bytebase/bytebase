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
      <BBAvatar :username="currentUser.name"> </BBAvatar>
    </button>
    <BBContextMenu
      ref="menu"
      class="origin-top-left absolute right-0 mt-2 w-36 rounded-md shadow-lg"
    >
      <div v-if="isDevOrDemo">
        <div v-if="currentUser.email != 'demo@example.com'" class="py-1">
          <a
            @click.prevent="switchToOwner"
            class="cursor-pointer block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
            role="menuitem"
          >
            Switch to Owner
          </a>
        </div>
        <div v-if="currentUser.email != 'jerry@example.com'" class="py-1">
          <a
            @click.prevent="switchToDBA"
            class="cursor-pointer block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
            role="menuitem"
          >
            Switch to DBA
          </a>
        </div>
        <div v-if="currentUser.email != 'tom@example.com'" class="py-1">
          <a
            @click.prevent="switchToDeveloper"
            class="cursor-pointer block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
            role="menuitem"
          >
            Switch to Dev
          </a>
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
          class="cursor-pointer block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
          role="menuitem"
          >Logout</a
        >
      </div>
    </BBContextMenu>
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useStore } from "vuex";

export default {
  name: "ProfileDropdown",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

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

    const switchToOwner = () => {
      store.dispatch("auth/login", {
        username: "demo@example.com",
        password: "1024",
      });
    };

    const switchToDBA = () => {
      store.dispatch("auth/login", {
        username: "jerry@example.com",
        password: "aaa",
      });
    };

    const switchToDeveloper = () => {
      store.dispatch("auth/login", {
        username: "tom@example.com",
        password: "aaa",
      });
    };

    return {
      currentUser,
      logout,
      switchToOwner,
      switchToDBA,
      switchToDeveloper,
    };
  },
};
</script>
