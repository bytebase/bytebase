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
      <PrincipalAvatar :principal="currentUser" />
    </button>
    <BBContextMenu ref="menu" class="origin-top-left mt-2 w-48">
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
      <div v-if="!isRelease" class="md:hidden py-1">
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
      <div
        v-if="!isRelease"
        @click.prevent="ping"
        class="py-1 menu-item"
        role="menuitem"
      >
        Ping
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
import PrincipalAvatar from "./PrincipalAvatar.vue";
import { ServerInfo } from "../types";

export default {
  name: "ProfileDropdown",
  components: { PrincipalAvatar },
  props: {},
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const logout = () => {
      store.dispatch("auth/logout").then(() => {
        router.push({ name: "auth.signin" });
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
        key: "project.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "database.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "schema.update",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "member.addOrInvite",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "hidden",
        newState: false,
      });

      store.dispatch("uistate/saveIntroStateByKey", {
        key: "guide.environment",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "guide.instance",
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

    const ping = () => {
      store.dispatch("actuator/info").then((info: ServerInfo) => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "SUCCESS",
          title: info,
        });
      });
    };

    return {
      currentUser,
      resetQuickstart,
      logout,
      switchToOwner,
      switchToDBA,
      switchToDeveloper,
      ping,
    };
  },
};
</script>
