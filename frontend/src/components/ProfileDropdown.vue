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
            {{ $t("common.role-switch.owner") }}
          </a>
        </div>
        <div v-if="currentUser.role != 'DBA'" class="py-1">
          <a class="menu-item" role="menuitem" @click.prevent="switchToDBA">
            {{ $t("common.role-switch.dba") }}
          </a>
        </div>
        <div v-if="currentUser.email != 'DEVELOPER'" class="py-1">
          <a
            class="menu-item"
            role="menuitem"
            @click.prevent="switchToDeveloper"
          >
            {{ $t("common.role-switch.developer") }}
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
        <div
          class="menu-item relative"
          @mouseenter="$refs.languageMenu.toggle($event)"
          @mouseleave="$refs.languageMenu.toggle($event)"
          @click.capture.prevent="$refs.languageMenu.toggle($event)"
        >
          <span>{{ $t("common.language") }}</span>
          <BBContextMenu
            ref="languageMenu"
            class="origin-left absolute left-0 top-0 -translate-x-48 transform"
          >
            <div
              class="menu-item px-3 py-1 hover:bg-gray-100"
              :class="{ 'bg-gray-100': locale === 'en' }"
              @click.prevent="toggleLocale('en')"
            >
              <div class="radio text-sm">
                <input type="radio" class="btn" :checked="locale === 'en'" />
                <label class="ml-2">English</label>
              </div>
            </div>
            <div
              class="menu-item px-3 py-1 hover:bg-gray-100"
              :class="{ 'bg-gray-100': locale === 'zh-CN' }"
              @click.prevent="toggleLocale('zh-CN')"
            >
              <div class="radio text-sm">
                <input type="radio" class="btn" :checked="locale === 'zh-CN'" />
                <label class="ml-2">简体中文</label>
              </div>
            </div>
          </BBContextMenu>
        </div>
        <a
          v-if="showQuickstart"
          class="menu-item"
          role="menuitem"
          @click.prevent="resetQuickstart"
          >{{ $t("common.quickstart") }}</a
        >
        <a href="https://bytebase.com/docs" target="_blank" class="menu-item">
          {{ $t("common.help") }}
        </a>
      </div>
      <div class="border-t border-gray-100"></div>
      <div v-if="allowToggleDebug" class="py-1 menu-item">
        <div class="flex flex-row items-center space-x-2 justify-between">
          <span> Debug </span>
          <BBSwitch :value="isDebug" @toggle="switchDebug" />
        </div>
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
import { computed, defineComponent, ref } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import PrincipalAvatar from "./PrincipalAvatar.vue";
import { ServerInfo } from "../types";
import { isDBAOrOwner } from "../utils";
import { useLanguage } from "../composables/useLanguage";
import {
  pushNotification,
  useActuatorStore,
  useDebugStore,
  useUIStateStore,
} from "@/store";
import { storeToRefs } from "pinia";

export default defineComponent({
  name: "ProfileDropdown",
  components: { PrincipalAvatar },
  props: {},
  setup() {
    const store = useStore();
    const actuatorStore = useActuatorStore();
    const debugStore = useDebugStore();
    const uiStateStore = useUIStateStore();
    const router = useRouter();
    const { setLocale, locale } = useLanguage();
    const languageMenu = ref();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    // For now, debug mode is a global setting and will affect all users.
    // So we only allow DBA and Owner to toggle it.
    const allowToggleDebug = computed(() => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const showQuickstart = computed(() => !actuatorStore.isDemo);

    const logout = () => {
      store.dispatch("auth/logout").then(() => {
        router.push({ name: "auth.signin" });
      });
    };

    const resetQuickstart = () => {
      const keys = [
        "general.overview",
        "bookmark.create",
        "comment.create",
        "project.visit",
        "environment.visit",
        "instance.visit",
        "database.visit",
        "member.addOrInvite",
        "hidden",
        "guide.project",
        "guide.environment",
        "guide.instance",
        "guide.database",
        "kbar.open",
      ];
      keys.forEach((key) => {
        uiStateStore.saveIntroStateByKey({
          key,
          newState: false,
        });
      });
    };

    const switchToOwner = () => {
      store.dispatch("auth/login", {
        payload: {
          email: "demo@example.com",
          password: "1024",
        },
      });
    };

    const switchToDBA = () => {
      store.dispatch("auth/login", {
        payload: {
          email: "jerry@example.com",
          password: "aaa",
        },
      });
    };

    const switchToDeveloper = () => {
      store.dispatch("auth/login", {
        payload: {
          email: "tom@example.com",
          password: "aaa",
        },
      });
    };

    const { isDebug } = storeToRefs(debugStore);

    const switchDebug = () => {
      debugStore.patchDebug({
        isDebug: !isDebug.value,
      });
    };

    const ping = () => {
      actuatorStore.fetchInfo().then((info: ServerInfo) => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: info,
        });
      });
    };

    const toggleLocale = (lang: string) => {
      setLocale(lang);
      languageMenu.value.toggle();
    };

    return {
      currentUser,
      allowToggleDebug,
      showQuickstart,
      resetQuickstart,
      logout,
      switchToOwner,
      switchToDBA,
      switchToDeveloper,
      isDebug,
      switchDebug,
      ping,
      toggleLocale,
      languageMenu,
      locale,
    };
  },
});
</script>
