<template>
  <div class="flex items-center justify-between h-16">
    <div class="flex items-center h-full">
      <BytebaseLogo class="h-full overflow-y-hidden" />

      <div class="hidden sm:block">
        <div class="ml-4 flex items-baseline space-x-1">
          <router-link
            to="/sql-editor"
            class="router-link"
            exact-active-class="anchor-link"
            >{{ $t("sql-editor.self") }}</router-link
          >
          <router-link
            to="/sheets/my"
            class="router-link"
            exact-active-class="anchor-link"
            >{{ $t("sheet.self") }}</router-link
          >
          <router-link
            v-if="showAuditLogItem"
            :to="`/setting/audit-log?type=${sqlQueryAction}`"
            class="router-link"
            exact-active-class="anchor-link"
            >{{ $t("settings.sidebar.audit-log") }}</router-link
          >
        </div>
      </div>
    </div>
    <div>
      <div class="flex items-center space-x-3">
        <router-link to="/inbox">
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
        <div class="ml-2 -mr-2 flex sm:hidden">
          <!-- Mobile menu button -->
          <button
            class="icon-link inline-flex items-center justify-center rounded-md"
            @click.prevent="state.showMobileMenu = !state.showMobileMenu"
          >
            <span class="sr-only">Open main menu</span>
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
  <div v-if="state.showMobileMenu" class="block md:hidden space-y-1 pb-1">
    <router-link
      to="/sql-editor"
      class="bar-link rounded-md block px-3 py-2"
      exact-active-class="anchor-link"
    >
      {{ $t("sql-editor.self") }}
    </router-link>
    <router-link
      to="/sheets/my"
      class="bar-link rounded-md block px-3 py-2"
      exact-active-class="anchor-link"
    >
      {{ $t("sheet.self") }}
    </router-link>
    <router-link
      v-if="showAuditLogItem"
      :to="`/setting/audit-log?type=${sqlQueryAction}`"
      class="bar-link rounded-md block px-3 py-2"
    >
      {{ $t("settings.sidebar.audit-log") }}
    </router-link>
  </div>
</template>

<script lang="ts">
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useLocalStorage } from "@vueuse/core";
import { computed, reactive, watchEffect, defineComponent } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import ProfileBrandingLogo from "@/components/ProfileBrandingLogo.vue";
import ProfileDropdown from "@/components/ProfileDropdown.vue";
import { useCurrentUser, useCurrentUserV1, useInboxV1Store } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { UNKNOWN_ID } from "@/types";
import { LogEntity_Action } from "@/types/proto/v1/logging_service";
import { hasWorkspacePermissionV1 } from "@/utils";

interface LocalState {
  showMobileMenu: boolean;
}

export default defineComponent({
  name: "EditorHeader",
  components: { BytebaseLogo, ProfileBrandingLogo, ProfileDropdown },
  setup() {
    const { t, availableLocales, locale } = useI18n();
    const inboxV1Store = useInboxV1Store();
    const settingV1Store = useSettingV1Store();
    const router = useRouter();

    const state = reactive<LocalState>({
      showMobileMenu: false,
    });

    const currentUser = useCurrentUser();
    const currentUserV1 = useCurrentUserV1();

    const logoUrl = computed((): string | undefined => {
      return settingV1Store.brandingLogo;
    });

    const showAuditLogItem = computed((): boolean => {
      return hasWorkspacePermissionV1(
        "bb.permission.workspace.audit-log",
        currentUserV1.value.userRole
      );
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
        id: "bb.navigation.inbox",
        name: "Inbox",
        shortcut: ["g", "m"],
        section: t("kbar.navigation"),
        keywords: "navigation",
        perform: () => router.push({ name: "setting.inbox" }),
      }),
    ]);
    useRegisterActions(kbarActions);

    const storage = useLocalStorage("bytebase_options", {}) as any;

    const setLocale = (lang: string) => {
      locale.value = lang;
      storage.value = {
        appearance: {
          language: lang,
        },
      };
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

    const goBack = () => {
      if (window.history.state?.back) {
        router.go(-1);
      } else {
        router.push("/");
      }
    };

    return {
      state,
      showAuditLogItem,
      inboxSummary,
      logoUrl,
      goBack,
      sqlQueryAction: LogEntity_Action.ACTION_DATABASE_SQL_EDITOR_QUERY,
    };
  },
});
</script>

<style scoped>
.router-link {
  @apply text-base ml-2 truncate px-2 py-2 rounded-lg no-underline hover:bg-gray-200;
}

.router-link-active {
  @apply bg-gray-200;
}
</style>
