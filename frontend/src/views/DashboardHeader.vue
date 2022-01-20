<template>
  <div class="flex items-center justify-between h-16">
    <div class="flex items-center">
      <div class="flex-shrink-0 w-44">
        <router-link to="/" class="select-none" active-class exact-active-class>
          <img
            class="h-12 w-auto"
            src="../assets/logo-full.svg"
            alt="Bytebase"
          />
        </router-link>
      </div>
      <div class="hidden sm:block">
        <div class="ml-6 flex items-baseline space-x-1 whitespace-nowrap">
          <router-link
            v-if="showDBAItem"
            to="/issue"
            class="bar-link px-2 py-2 rounded-md"
            >{{ $t("common.issues") }}</router-link
          >

          <router-link to="/project" class="bar-link px-2 py-2 rounded-md">
            {{ $t("common.projects") }}
          </router-link>

          <router-link to="/db" class="bar-link px-2 py-2 rounded-md">{{
            $t("common.databases")
          }}</router-link>

          <router-link to="/instance" class="bar-link px-2 py-2 rounded-md">{{
            $t("common.instances")
          }}</router-link>

          <router-link
            to="/environment"
            class="bar-link px-2 py-2 rounded-md"
            >{{ $t("common.environments") }}</router-link
          >
          <router-link
            to="/setting/member"
            class="bar-link px-2 py-2 rounded-md"
            >{{ $t("common.settings") }}</router-link
          >
          <router-link
            v-if="isDevFeatures"
            to="/sql-editor"
            class="bar-link px-2 py-2 rounded-md"
            >{{ $t("sql-editor.self") }}</router-link
          >
        </div>
      </div>
    </div>
    <div>
      <div class="flex items-center space-x-3">
        <div
          v-if="isDevFeatures"
          class="hidden md:flex sm:flex-row items-center space-x-2 text-sm font-medium"
        >
          <span class="hidden lg:block font-normal text-accent">
            {{ $t("setting.plan.self") }}
          </span>
          <div
            class="bar-link"
            :class="currentPlan == 0 ? 'underline' : ''"
            @click.prevent="switchToFree"
          >
            {{ $t("setting.plan.free") }}
          </div>
          <div
            class="bar-link"
            :class="currentPlan == 1 ? 'underline' : ''"
            @click.prevent="switchToTeam"
          >
            {{ $t("setting.plan.team") }}
          </div>
          <!-- <div
            class="bar-link"
            :class="currentPlan == 2 ? 'underline' : ''"
            @click.prevent="switchToEnterprise"
          >
            {{ $t('setting.plan.enterprise') }}
          </div>-->
        </div>
        <div
          v-if="!isRelease"
          class="hidden md:flex sm:flex-row items-center space-x-2 text-sm font-medium"
        >
          <span class="hidden lg:block font-normal text-accent">
            {{ $t("settings.profile.role") }}
          </span>
          <div
            class="bar-link"
            :class="currentUser.role == 'OWNER' ? 'underline' : ''"
            @click.prevent="switchToOwner"
          >
            {{ $t("common.role.owner") }}
          </div>
          <div
            class="bar-link"
            :class="currentUser.role == 'DBA' ? 'underline' : ''"
            @click.prevent="switchToDBA"
          >
            {{ $t("common.role.dba") }}
          </div>
          <div
            class="bar-link"
            :class="currentUser.role == 'DEVELOPER' ? 'underline' : ''"
            @click.prevent="switchToDeveloper"
          >
            {{ $t("common.role.developer") }}
          </div>
        </div>
        <router-link to="/inbox" exact-active-class>
          <span
            v-if="inboxSummary.hasUnread"
            class="absolute rounded-full ml-4 -mt-1 h-2.5 w-2.5 bg-accent opacity-75"
          ></span>
          <heroicons-outline:bell class="w-6 h-6" />
        </router-link>
        <div v-if="isDevFeatures" class="cursor-pointer" @click="toggleLocales">
          <heroicons-outline:translate class="w-6 h-6" />
        </div>
        <div class="ml-2">
          <ProfileDropdown />
        </div>
        <div class="ml-2 -mr-2 flex sm:hidden">
          <!-- Mobile menu button -->
          <button
            class="icon-link inline-flex items-center justify-center rounded-md"
            @click.prevent="state.showMobileMenu = !state.showMobileMenu"
          >
            <span class="sr-only">Open main menu</span>
            <!--
              Heroicon name: menu

              Menu open: "hidden", Menu closed: "block"
            -->
            <heroicons-solid:dots-vertical class="w-6 h-6" />
          </button>
        </div>
      </div>
    </div>
  </div>

  <!--
      Mobile menu, toggle classes based on menu state.

      Open: "block", closed: "hidden"
  -->
  <div v-if="state.showMobileMenu" class="block md:hidden">
    <router-link
      v-if="showDBAItem"
      to="/issue"
      class="bar-link rounded-md block px-3 py-2"
    >
      {{ $t("common.issues") }}
    </router-link>

    <router-link to="/project" class="bar-link rounded-md block px-3 py-2">
      {{ $t("common.projects") }}
    </router-link>

    <router-link to="/db" class="bar-link rounded-md block px-3 py-2">
      {{ $t("common.databases") }}
    </router-link>

    <router-link to="/instance" class="bar-link rounded-md block px-3 py-2">{{
      $t("common.instances")
    }}</router-link>

    <router-link
      to="/environment"
      class="bar-link rounded-md block px-3 py-2"
      >{{ $t("common.environments") }}</router-link
    >

    <router-link
      to="/setting/member"
      class="bar-link rounded-md block px-3 py-2"
      >{{ $t("common.settings") }}</router-link
    >
  </div>
</template>

<script lang="ts">
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { computed, reactive, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { useStore } from "vuex";
import { useI18n } from "vue-i18n";

import ProfileDropdown from "../components/ProfileDropdown.vue";
import { InboxSummary, PlanType, UNKNOWN_ID } from "../types";
import { isDBAOrOwner, isDev } from "../utils";
import { useLanguage } from "../composables/useLanguage";

interface LocalState {
  showMobileMenu: boolean;
}

export default {
  name: "DashboardHeader",
  components: { ProfileDropdown },
  setup() {
    const { t, availableLocales } = useI18n();
    const store = useStore();
    const router = useRouter();
    const { setLocale, toggleLocales } = useLanguage();

    const state = reactive<LocalState>({
      showMobileMenu: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const currentPlan = computed(() => store.getters["plan/currentPlan"]());

    const showDBAItem = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const isDevFeatures = computed((): boolean => {
      return isDev();
    });

    const prepareInboxSummary = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store.dispatch("inbox/fetchInboxSummaryByUser", currentUser.value.id);
      }
    };

    watchEffect(prepareInboxSummary);

    const inboxSummary = computed((): InboxSummary => {
      return store.getters["inbox/inboxSummaryByUser"](currentUser.value.id);
    });

    const switchToOwner = () => {
      store.dispatch("auth/login", {
        authProvider: "BYTEBASE",
        payload: {
          email: "demo@example.com",
          password: "1024",
        },
      });
    };

    const switchToDBA = () => {
      store.dispatch("auth/login", {
        authProvider: "BYTEBASE",
        payload: {
          email: "jerry@example.com",
          password: "2048",
        },
      });
    };

    const switchToDeveloper = () => {
      store.dispatch("auth/login", {
        authProvider: "BYTEBASE",
        payload: {
          email: "tom@example.com",
          password: "4096",
        },
      });
    };

    const switchToFree = () => {
      store.dispatch("plan/changePlan", PlanType.FREE);
    };

    const switchToTeam = () => {
      store.dispatch("plan/changePlan", PlanType.TEAM);
    };

    const switchToEnterprise = () => {
      store.dispatch("plan/changePlan", PlanType.ENTERPRISE);
    };

    const kbarActions = computed(() => [
      defineAction({
        id: "bb.navigation.projects",
        name: "Projects",
        shortcut: ["g", "p"],
        section: t("kbar.navigation"),
        keywords: "navigation",
        perform: () => router.push({ name: "workspace.project" }),
      }),
      defineAction({
        id: "bb.navigation.databases",
        name: "Databases",
        shortcut: ["g", "d"],
        section: t("kbar.navigation"),
        keywords: "navigation db",
        perform: () => router.push({ name: "workspace.database" }),
      }),
      defineAction({
        id: "bb.navigation.instances",
        name: "Instances",
        shortcut: ["g", "i"],
        section: t("kbar.navigation"),
        keywords: "navigation",
        perform: () => router.push({ name: "workspace.instance" }),
      }),
      defineAction({
        id: "bb.navigation.environments",
        name: "Environments",
        shortcut: ["g", "e"],
        section: t("kbar.navigation"),
        keywords: "navigation",
        perform: () => router.push({ name: "workspace.environment" }),
      }),
      defineAction({
        id: "bb.navigation.settings",
        name: "Settings",
        shortcut: ["g", "s"],
        section: t("kbar.navigation"),
        keywords: "navigation",
        perform: () => router.push({ name: "setting.workspace.member" }),
      }),
      defineAction({
        id: "bb.navigation.inbox",
        name: "Inbox",
        shortcut: ["g", "m"],
        section: t("kbar.navigation"),
        keywords: "navigation",
        perform: () => router.push({ name: "setting.inbox" }),
      }),
    ]);
    useRegisterActions(kbarActions);

    const I18N_CHANGE_ACTION_ID_NAMESPACE = "bb.preferences.locale";
    const i18nChangeAction = computed(() =>
      defineAction({
        // here `id` is "bb.preferences.locale"
        id: I18N_CHANGE_ACTION_ID_NAMESPACE,
        section: t("kbar.preferences.common"),
        name: t("kbar.preferences.change-language"),
        keywords: "language lang locale",
      })
    );
    const i18nActions = computed(() => [
      i18nChangeAction.value,
      ...availableLocales.map((lang) => {
        return defineAction({
          // here `id` looks like "bb.preferences.locale.en"
          id: `${I18N_CHANGE_ACTION_ID_NAMESPACE}.${lang}`,
          name: lang,
          parent: I18N_CHANGE_ACTION_ID_NAMESPACE,
          keywords: `language lang locale ${lang}`,
          perform: () => setLocale(lang),
        });
      }),
    ]);
    useRegisterActions(i18nActions);

    return {
      state,
      currentUser,
      currentPlan,
      showDBAItem,
      isDevFeatures,
      inboxSummary,
      switchToOwner,
      switchToDBA,
      switchToDeveloper,
      switchToFree,
      switchToTeam,
      switchToEnterprise,
      toggleLocales,
    };
  },
};
</script>
