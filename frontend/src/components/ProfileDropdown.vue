<template>
  <div class="relative">
    <button
      id="user-menu"
      type="button"
      class="flex text-sm text-white focus:outline-none focus:shadow-solid"
      aria-label="User menu"
      aria-haspopup="true"
      @click.prevent="menu.toggle()"
      @contextmenu.capture.prevent="menu.toggle()"
    >
      <UserAvatar :user="currentUserV1" />
    </button>
    <BBContextMenu ref="menu" class="origin-top-left mt-2 w-48">
      <router-link
        class="px-4 py-3 menu-item"
        to="/setting/profile"
        role="menuitem"
      >
        <p class="text-sm flex justify-between">
          <span class="text-main font-medium truncate">
            {{ currentUserV1.title }}
          </span>
          <span class="text-control">
            {{ roleNameV1(currentUserV1.userRole) }}
          </span>
        </p>
        <p class="text-sm text-control truncate">
          {{ currentUserV1.email }}
        </p>
      </router-link>
      <div class="border-t border-gray-100"></div>
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
          @mouseenter="languageMenu.toggle()"
          @mouseleave="languageMenu.toggle()"
          @click.capture.prevent="languageMenu.toggle()"
        >
          <span>{{ $t("common.language") }}</span>
          <BBContextMenu
            ref="languageMenu"
            class="origin-left absolute left-0 -top-1 -translate-x-48 transform"
          >
            <div
              class="menu-item px-3 py-1 hover:bg-gray-100"
              :class="{ 'bg-gray-100': locale === 'en-US' }"
              @click.prevent="toggleLocale('en-US')"
            >
              <div class="radio text-sm">
                <input type="radio" class="btn" :checked="locale === 'en-US'" />
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
            <div
              class="menu-item px-3 py-1 hover:bg-gray-100"
              :class="{ 'bg-gray-100': locale === 'es-ES' }"
              @click.prevent="toggleLocale('es-ES')"
            >
              <div class="radio text-sm">
                <input type="radio" class="btn" :checked="locale === 'es-ES'" />
                <label class="ml-2">Español</label>
              </div>
            </div>
          </BBContextMenu>
        </div>
        <a
          v-if="showQuickstart"
          class="menu-item"
          role="menuitem"
          @click.prevent="resetQuickstart"
          >{{ $t("quick-start.self") }}</a
        >
        <a
          href="https://bytebase.com/docs?source=console"
          target="_blank"
          class="menu-item"
        >
          {{ $t("common.help") }}
        </a>
      </div>
      <div class="border-t border-gray-100"></div>
      <div v-if="allowToggleDebug" class="py-1 menu-item">
        <div class="flex flex-row items-center space-x-2 justify-between">
          <span>Debug</span>
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

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { useLanguage } from "@/composables/useLanguage";
import {
  pushNotification,
  useActuatorV1Store,
  useAuthStore,
  useCurrentUserV1,
  useUIStateStore,
} from "@/store";
import { hasWorkspacePermissionV1, roleNameV1 } from "@/utils";
import UserAvatar from "./User/UserAvatar.vue";

const actuatorStore = useActuatorV1Store();
const authStore = useAuthStore();
const uiStateStore = useUIStateStore();
const router = useRouter();
const { setLocale, locale } = useLanguage();
const menu = ref();
const languageMenu = ref();
const currentUserV1 = useCurrentUserV1();

// For now, debug mode is a global setting and will affect all users.
// So we only allow DBA and Owner to toggle it.
const allowToggleDebug = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.debug",
    currentUserV1.value.userRole
  );
});

const showQuickstart = computed(() => !actuatorStore.isDemo);

const logout = () => {
  authStore.logout().then(() => {
    router.push({ name: "auth.signin" });
  });
};

const resetQuickstart = () => {
  const keys = [
    "hidden",
    "issue.visit",
    "project.visit",
    "environment.visit",
    "instance.visit",
    "database.visit",
    "member.addOrInvite",
    "kbar.open",
    "data.query",
    "help.issue.detail",
    "help.project",
    "help.environment",
    "help.instance",
    "help.database",
    "help.member",
  ];
  keys.forEach((key) => {
    uiStateStore.saveIntroStateByKey({
      key,
      newState: false,
    });
  });
};

const { isDebug } = storeToRefs(actuatorStore);

const switchDebug = () => {
  actuatorStore.patchDebug({
    debug: !isDebug.value,
  });
};

const ping = () => {
  actuatorStore.fetchServerInfo().then((info) => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: JSON.stringify(info, null, 4),
    });
  });
};

const toggleLocale = (lang: string) => {
  setLocale(lang);
  languageMenu.value.toggle();
};
</script>
