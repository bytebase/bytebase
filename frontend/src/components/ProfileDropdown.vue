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
      class="origin-top-left absolute right-0 mt-2 w-48 rounded-md shadow-lg"
    >
      <router-link
        class="px-4 py-3 menu-item"
        :to="`/u/${currentUser.id}`"
        role="menuitem"
      >
        <p class="text-sm text-main font-medium">
          {{ currentUser.name }}
        </p>
        <p class="text-sm text-control truncate">
          {{ currentUser.email }}
        </p>
      </router-link>
      <div class="border-t border-gray-100"></div>
      <div v-if="isDevOrDemo" class="md:hidden py-1">
        <div v-if="currentUser.role != 'OWNER'" class="py-1">
          <a @click.prevent="switchToOwner" class="menu-item" role="menuitem">
            Switch to Owner
          </a>
        </div>
        <div v-if="currentUser.role != 'DBA'" class="py-1">
          <a @click.prevent="switchToDBA" class="menu-item" role="menuitem">
            Switch to DBA
          </a>
        </div>
        <div v-if="currentUser.email != 'DEVELOPER'" class="py-1">
          <a
            @click.prevent="switchToDeveloper"
            class="menu-item"
            role="menuitem"
          >
            Switch to Developer
          </a>
        </div>
      </div>
      <div class="border-t border-gray-100"></div>
      <div class="py-1">
        <router-link to="/setting" class="menu-item" role="menuitem"
          >Settings</router-link
        >
        <a @click.prevent="resetQuickstart" class="menu-item" role="menuitem"
          >Reset Quickstart</a
        >
      </div>
      <div class="border-t border-gray-100"></div>
      <div class="py-1">
        <a href="#" class="menu-item" role="menuitem">Changelog</a>
        <a
          href="https://github.com/bytebase/bytebase/discussions"
          target="_blank"
          class="menu-item"
          role="menuitem"
          >Support</a
        >
      </div>
      <div class="border-t border-gray-100"></div>
      <div class="py-1">
        <a @click.prevent="logout" class="menu-item" role="menuitem">Logout</a>
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
          router.push({ name: "auth.signin" });
        })
        .catch((error: Error) => {
          console.log(error);
          return;
        });
    };

    const resetQuickstart = () => {
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "bookmark.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "environment.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "instance.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "database.request",
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

    const isOwner = computed(() => {
      return currentUser.value.role == "OWNER";
    });

    const isDBA = computed(() => {
      return currentUser.value.role == "DBA";
    });

    const isDeveloper = computed(() => {
      return currentUser.value.role == "DEVELOPER";
    });

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
      isOwner,
      isDBA,
      isDeveloper,
      switchToOwner,
      switchToDBA,
      switchToDeveloper,
    };
  },
};
</script>
