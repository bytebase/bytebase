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
          class="cursor-pointer block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
          role="menuitem"
          >Settings</router-link
        >
        <a
          @click.prevent="resetQuickstart"
          class="cursor-pointer block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
          role="menuitem"
          >Reset Quickstart</a
        >
      </div>
      <div class="border-t border-gray-100"></div>
      <div class="py-1">
        <a
          href="#"
          class="cursor-pointer block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
          role="menuitem"
          >Changelog</a
        >
        <a
          href="https://github.com/bytebase/bytebase/discussions"
          target="_blank"
          class="cursor-pointer block px-4 py-2 text-sm leading-5 text-gray-700 hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus:bg-gray-100 focus:text-gray-900"
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
import { useRouter } from "vue-router";

export default {
  name: "ProfileDropdown",
  props: {},
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const logout = () => {
      store
        .dispatch("auth/logout")
        .then(() => {
          // If using router.push, looks like hook reactive to currentUser would get called first.
          // So we just do a reload here.
          location.reload();
        })
        .catch((error: Error) => {
          console.log(error);
          return;
        });
    };

    const resetQuickstart = () => {
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "environment.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "instance.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "datasource.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "table.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "member.invite",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "hidden",
        newState: false,
      });
    };

    const switchToOwner = () => {
      store.dispatch("auth/login", {
        email: "demo@example.com",
        password: "1024",
      });
    };

    const switchToDBA = () => {
      store.dispatch("auth/login", {
        email: "jerry@example.com",
        password: "aaa",
      });
    };

    const switchToDeveloper = () => {
      store.dispatch("auth/login", {
        email: "tom@example.com",
        password: "aaa",
      });
    };

    return {
      currentUser,
      resetQuickstart,
      logout,
      switchToOwner,
      switchToDBA,
      switchToDeveloper,
    };
  },
};
</script>
