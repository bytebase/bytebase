<template>
  <div class="flex items-center justify-between h-16 pl-2 pr-4">
    <div class="flex items-center">
      <BytebaseLogo class="block md:hidden" />

      <div class="hidden md:block">
        <div class="flex items-baseline space-x-1 whitespace-nowrap">
          <router-link
            v-if="shouldShowIssueEntry"
            to="/issue"
            class="bar-link px-2 py-2 rounded-md"
            :class="getRouteLinkClass('/issue')"
            >{{ $t("common.issues") }}</router-link
          >

          <router-link
            to="/project"
            class="bar-link px-2 py-2 rounded-md"
            :class="getRouteLinkClass('/project')"
            data-label="bb-header-project-button"
          >
            {{ $t("common.projects") }}
          </router-link>

          <router-link
            to="/db"
            class="bar-link px-2 py-2 rounded-md"
            :class="getRouteLinkClass('/db')"
            data-label="bb-dashboard-header-database-entry"
            >{{ $t("common.databases") }}</router-link
          >

          <router-link
            v-if="shouldShowInstanceEntry"
            to="/instance"
            class="bar-link px-2 py-2 rounded-md"
            :class="getRouteLinkClass('/instance')"
            >{{ $t("common.instances") }}</router-link
          >

          <router-link
            to="/environment"
            class="bar-link px-2 py-2 rounded-md"
            :class="getRouteLinkClass('/environment')"
            >{{ $t("common.environments") }}</router-link
          >
          <router-link
            to="/setting/member"
            class="bar-link px-2 py-2 rounded-md"
            :class="getRouteLinkClass('/setting')"
            >{{ $t("common.settings") }}</router-link
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
          <div
            class="cursor-pointer hover:bg-link-hover focus:outline-none"
            :class="
              currentPlan == PlanType.FREE
                ? 'underline text-accent'
                : 'text-main'
            "
            @click.prevent="switchToFree"
          >
            {{ $t("subscription.plan.free.title") }}
          </div>
          <div
            class="cursor-pointer hover:bg-link-hover focus:outline-none"
            :class="
              currentPlan == PlanType.TEAM
                ? 'underline text-accent'
                : 'text-main'
            "
            @click.prevent="switchToTeam"
          >
            {{ $t("subscription.plan.team.title") }}
          </div>
          <div
            class="cursor-pointer hover:bg-link-hover focus:outline-none"
            :class="
              currentPlan == PlanType.ENTERPRISE
                ? 'underline text-accent'
                : 'text-main'
            "
            @click.prevent="switchToEnterprise"
          >
            {{ $t("subscription.plan.enterprise.title") }}
          </div>
          <div class="cursor-pointer" @click="toggleLocales">
            <heroicons-outline:translate class="w-6 h-6" />
          </div>
          <div class="cursor-pointer" @click="toggleDebug">
            <svg
              class="w-6 h-6"
              :class="isDebug ? `text-accent` : `text-gray-400`"
              fill="currentColor"
              viewBox="0 0 32 32"
            >
              <path
                d="M29,15h-5.1c-0.1-1.2-0.5-2.4-1-3.5c1.9-1.5,3.1-3.7,3.1-6.1V5c0-0.6-0.4-1-1-1s-1,0.4-1,1v0.4c0,1.8-0.8,3.4-2.2,4.5  c-0.5-0.7-1.2-1.2-1.9-1.7c0-0.1,0-0.1,0-0.2c0-2.2-1.8-4-4-4s-4,1.8-4,4c0,0.1,0,0.1,0,0.2c-0.7,0.5-1.3,1-1.9,1.7  C8.8,8.8,8,7.2,8,5.4V5c0-0.6-0.4-1-1-1S6,4.4,6,5v0.4c0,2.4,1.1,4.7,3.1,6.1c-0.5,1-0.9,2.2-1,3.5H3c-0.6,0-1,0.4-1,1s0.4,1,1,1  h5.1c0.1,1.2,0.5,2.4,1,3.5C7.1,21.9,6,24.2,6,26.6V27c0,0.6,0.4,1,1,1s1-0.4,1-1v-0.4c0-1.8,0.8-3.4,2.2-4.5  c1.5,1.8,3.5,2.9,5.8,2.9s4.4-1.1,5.8-2.9c1.4,1.1,2.2,2.7,2.2,4.5V27c0,0.6,0.4,1,1,1s1-0.4,1-1v-0.4c0-2.4-1.1-4.7-3.1-6.1  c0.5-1,0.9-2.2,1-3.5H29c0.6,0,1-0.4,1-1S29.6,15,29,15z"
                stroke-width="1"
              />
            </svg>
          </div>
        </div>
        <div
          v-if="currentPlan === PlanType.FREE"
          class="flex justify-between items-center min-w-fit px-4 py-2 bg-emerald-500 text-sm font-medium text-white rounded-md cursor-pointer"
          @click="handleWantHelp"
        >
          <span class="hidden lg:block mr-2">{{ $t("common.want-help") }}</span>
          <heroicons-outline:chat-bubble-left-right class="w-5 h-5" />
        </div>
        <router-link to="/inbox" exact-active-class="">
          <span
            v-if="inboxSummary.unread > 0"
            class="absolute rounded-full ml-4 -mt-1 h-2.5 w-2.5 bg-accent opacity-75"
          ></span>
          <heroicons-outline:bell class="w-6 h-6" />
        </router-link>
        <div class="ml-2">
          <ProfileBrandingLogo>
            <ProfileDropdown />
          </ProfileBrandingLogo>
        </div>
        <div class="ml-2 -mr-2 flex md:hidden">
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
      v-if="shouldShowIssueEntry"
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
  <BBModal
    v-if="state.showQRCodeModal"
    :title="$t('common.want-help')"
    @close="state.showQRCodeModal = false"
  >
    <img
      class="w-48 h-48"
      src="@/assets/bb-helper-wechat-qrcode.webp"
      alt="bb_helper"
    />
  </BBModal>
</template>

<script lang="ts">
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { storeToRefs } from "pinia";
import { computed, reactive, watchEffect, defineComponent } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  useCurrentUser,
  useActuatorV1Store,
  useSubscriptionV1Store,
  useInboxV1Store,
  useCurrentUserV1,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import BytebaseLogo from "../components/BytebaseLogo.vue";
import ProfileBrandingLogo from "../components/ProfileBrandingLogo.vue";
import ProfileDropdown from "../components/ProfileDropdown.vue";
import { useLanguage } from "../composables/useLanguage";
import { UNKNOWN_ID } from "../types";
import { hasWorkspacePermissionV1, isDev } from "../utils";

interface LocalState {
  showMobileMenu: boolean;
  showQRCodeModal: boolean;
}

export default defineComponent({
  name: "DashboardHeader",
  components: {
    BytebaseLogo,
    ProfileBrandingLogo,
    ProfileDropdown,
  },
  setup() {
    const { t, availableLocales } = useI18n();
    const actuatorV1Store = useActuatorV1Store();
    const inboxV1Store = useInboxV1Store();
    const subscriptionStore = useSubscriptionV1Store();
    const router = useRouter();
    const route = useRoute();
    const { setLocale, toggleLocales, locale } = useLanguage();

    const state = reactive<LocalState>({
      showMobileMenu: false,
      showQRCodeModal: false,
    });

    const currentUser = useCurrentUser();
    const currentUserV1 = useCurrentUserV1();

    const { currentPlan } = storeToRefs(subscriptionStore);

    const getRouteLinkClass = (prefix: string): string[] => {
      const { path } = route;
      const isActiveRoute = path === prefix || path.startsWith(`${prefix}/`);
      const classes: string[] = [];
      if (isActiveRoute) {
        classes.push("router-link-active", "bg-link-hover");
      }
      return classes;
    };

    const shouldShowIssueEntry = computed((): boolean => {
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-issue",
        currentUserV1.value.userRole
      );
    });

    const shouldShowInstanceEntry = computed(() => {
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-instance",
        currentUserV1.value.userRole
      );
    });

    const isDevFeatures = computed((): boolean => {
      return isDev();
    });

    const prepareInboxSummary = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        inboxV1Store.fetchInboxSummary();
      }
    };

    watchEffect(prepareInboxSummary);

    const inboxSummary = computed(() => {
      return inboxV1Store.inboxSummary;
    });

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

    const switchToFree = () => {
      subscriptionStore.patchSubscription("");
    };

    const switchToTeam = () => {
      subscriptionStore.patchSubscription(
        import.meta.env.BB_DEV_TEAM_LICENSE as string
      );
    };

    const switchToEnterprise = () => {
      subscriptionStore.patchSubscription(
        import.meta.env.BB_DEV_ENTERPRISE_LICENSE as string
      );
    };

    const { isDebug } = storeToRefs(actuatorV1Store);

    const toggleDebug = () => {
      actuatorV1Store.patchDebug({
        debug: !isDebug.value,
      });
    };

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

    const handleWantHelp = () => {
      if (locale.value === "zh-CN") {
        state.showQRCodeModal = true;
      } else {
        window.open(
          "https://www.bytebase.com/docs/faq#how-to-reach-us",
          "_blank"
        );
      }
    };

    return {
      state,
      getRouteLinkClass,
      shouldShowInstanceEntry,
      shouldShowIssueEntry,
      currentPlan,
      PlanType,
      isDevFeatures,
      inboxSummary,
      switchToFree,
      switchToTeam,
      switchToEnterprise,
      isDebug,
      toggleDebug,
      toggleLocales,
      handleWantHelp,
    };
  },
});
</script>
