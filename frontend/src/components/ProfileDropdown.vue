<template>
  <div class="relative">
    <button
      id="user-menu"
      type="button"
      class="flex text-sm text-white focus:outline-none focus:shadow-solid"
      aria-label="User menu"
      aria-haspopup="true"
      @click.prevent="$refs.menu.toggle($event)"
      @contextmenu.capture.prevent="$refs.menu.toggle($event)"
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
          <a class="menu-item" role="menuitem" @click.prevent="switchToOwner">
            Switch to Owner
          </a>
        </div>
        <div v-if="currentUser.role != 'DBA'" class="py-1">
          <a class="menu-item" role="menuitem" @click.prevent="switchToDBA">
            Switch to DBA
          </a>
        </div>
        <div v-if="currentUser.email != 'DEVELOPER'" class="py-1">
          <a
            class="menu-item"
            role="menuitem"
            @click.prevent="switchToDeveloper"
          >
            Switch to Developer
          </a>
        </div>
      </div>
      <div
        v-if="!isRelease"
        class="py-1 menu-item"
        role="menuitem"
        @click.prevent="ping"
      >
        Ping
      </div>
      <div class="border-t border-gray-100"></div>
      <div class="py-1">
        <router-link to="/setting" class="menu-item" role="menuitem">{{
          $t("common.settings")
        }}</router-link>
        <a
          v-if="showQuickstart"
          class="menu-item"
          role="menuitem"
          @click.prevent="resetQuickstart"
          >{{ $t("common.quickstart") }}</a
        >
      </div>
      <div class="border-t border-gray-100"></div>
      <div class="py-1">
        <a class="menu-item" role="menuitem" @click.prevent="logout">{{
          $t("common.logout")
        }}</a>
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
  setup() {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const showQuickstart = computed(() => {
      return !store.getters["actuator/isDemo"]();
    });

    const logout = () => {
      store.dispatch("auth/logout").then(() => {
        router.push({ name: "auth.signin" });
      });
    };

    const resetQuickstart = () => {
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "general.overview",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "bookmark.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "comment.create",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "project.visit",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "environment.visit",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "instance.visit",
        newState: false,
      });
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "database.visit",
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
        key: "guide.project",
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
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "guide.database",
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
      store.dispatch("actuator/fetchInfo").then((info: ServerInfo) => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "SUCCESS",
          title: info,
        });
      });
    };

    return {
      currentUser,
      showQuickstart,
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
