<template>
  <NDropdown
    :show="showDropdown"
    :options="options"
    @clickoutside="showDropdown = false"
  >
    <UserAvatar
      class="cursor-pointer"
      :size="'SMALL'"
      :user="currentUserV1"
      @click="showDropdown = true"
    />
  </NDropdown>
</template>

<script lang="tsx" setup>
import type { DropdownOption } from "naive-ui";
import { NDropdown, NSwitch } from "naive-ui";
import { storeToRefs } from "pinia";
import { twMerge } from "tailwind-merge";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { useLanguage } from "@/composables/useLanguage";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import {
  useActuatorV1Store,
  useAppFeature,
  useAuthStore,
  useCurrentUserV1,
  useSubscriptionV1Store,
  useUIStateStore,
} from "@/store";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2, isDev, isSQLEditorRoute } from "@/utils";
import Version from "./misc/Version.vue";
import ProfilePreview from "./ProfilePreview.vue";
import UserAvatar from "./User/UserAvatar.vue";

const { t } = useI18n();

const props = defineProps<{
  link?: boolean;
}>();

const router = useRouter();
const actuatorStore = useActuatorV1Store();
const authStore = useAuthStore();
const subscriptionStore = useSubscriptionV1Store();
const uiStateStore = useUIStateStore();
const { setLocale, locale } = useLanguage();
const currentUserV1 = useCurrentUserV1();
const showDropdown = ref(false);
const hideHelp = useAppFeature("bb.feature.hide-help");

// For now, debug mode is a global setting and will affect all users.
// So we only allow DBA and Owner to toggle it.
const allowToggleDebug = computed(() => {
  return hasWorkspacePermissionV2("bb.settings.set");
});
const { currentPlan } = storeToRefs(subscriptionStore);

const logout = () => {
  authStore.logout();
};

const resetQuickstart = () => {
  const keys = [
    "hidden",
    "issue.visit",
    "project.visit",
    "environment.visit",
    "instance.visit",
    "database.visit",
    "member.visit",
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
  showDropdown.value = false;
};

const { isDebug } = storeToRefs(actuatorStore);

const switchDebug = () => {
  actuatorStore.patchDebug({
    debug: !isDebug.value,
  });
};

const { enabledNewLayout, toggleLayout } = useIssueLayoutVersion();

const toggleLocale = (lang: string) => {
  setLocale(lang);
  showDropdown.value = false;
};

const switchPlan = (license: string) => {
  subscriptionStore.patchSubscription(license);
  showDropdown.value = false;
};

const languageOptions = computed((): DropdownOption[] => {
  const languages = [
    {
      label: "English",
      value: "en-US",
    },
    {
      label: "简体中文",
      value: "zh-CN",
    },
    {
      label: "Español",
      value: "es-ES",
    },
    {
      label: "日本語",
      value: "ja-JP",
    },
    {
      label: "Tiếng việt",
      value: "vi-VN",
    },
  ];

  return languages.map((item) => {
    const classes: string[] = [
      "menu-item cursor-pointer px-3 py-1 hover:bg-gray-100 w-48",
    ];
    if (locale.value === item.value) {
      classes.push("bg-gray-100");
    }

    return {
      key: item.value,
      type: "render",
      render() {
        return (
          <div
            key={item.value}
            class={twMerge(classes)}
            onClick={() => toggleLocale(item.value)}
          >
            <div class="radio cursor-pointer text-sm">
              <input
                type="radio"
                class="btn"
                checked={locale.value === item.value}
              />
              <label class="ml-2 cursor-pointer">{item.label}</label>
            </div>
          </div>
        );
      },
    };
  });
});

const licenseOptions = computed((): DropdownOption[] => {
  const options = [
    {
      label: t("subscription.plan.free.title"),
      value: "",
      plan: PlanType.FREE,
    },
    {
      label: t("subscription.plan.team.title"),
      value: import.meta.env.BB_DEV_TEAM_LICENSE as string,
      plan: PlanType.TEAM,
    },
    {
      label: t("subscription.plan.enterprise.title"),
      value: import.meta.env.BB_DEV_ENTERPRISE_LICENSE as string,
      plan: PlanType.ENTERPRISE,
    },
  ];

  return options.map((item) => {
    const classes = ["menu-item px-3 py-1 hover:bg-gray-100 w-48"];
    if (item.plan === currentPlan.value) {
      classes.push("bg-gray-100");
    }

    return {
      key: item.plan,
      type: "render",
      render() {
        return (
          <div
            key={item.value}
            class={classes.join(" ")}
            onClick={() => switchPlan(item.value)}
          >
            <div class="radio cursor-pointer text-sm">
              <input
                type="radio"
                class="btn"
                checked={currentPlan.value === item.plan}
              />
              <label class="ml-2 cursor-pointer">{item.label}</label>
            </div>
          </div>
        );
      },
    };
  });
});

const options = computed((): DropdownOption[] => [
  {
    key: "profile",
    type: "render",
    render() {
      return (
        <ProfilePreview
          link={props.link}
          onClick={() => (showDropdown.value = false)}
        />
      );
    },
  },
  {
    key: "header-divider",
    type: "divider",
  },
  {
    key: "language",
    type: "render",
    render() {
      return (
        <NDropdown
          key="switch-language"
          options={languageOptions.value}
          placement="left-start"
        >
          <div class="menu-item">{t("common.language")}</div>
        </NDropdown>
      );
    },
  },
  {
    key: "license",
    type: "render",
    show: isDev(),
    render() {
      return (
        <NDropdown
          key="switch-license"
          options={licenseOptions.value}
          placement="left-start"
        >
          <div class="menu-item">{t("common.license")}</div>
        </NDropdown>
      );
    },
  },
  {
    key: "quick-start",
    type: "render",
    show: actuatorStore.quickStartEnabled,
    render() {
      return (
        <div class="menu-item" onClick={resetQuickstart}>
          {t("quick-start.self")}
        </div>
      );
    },
  },
  {
    key: "help",
    type: "render",
    show: !hideHelp.value,
    render() {
      return (
        <a
          class="menu-item"
          target="_blank"
          href="https://docs.bytebase.com/introduction/what-is-bytebase/?source=console"
        >
          {t("common.help")}
        </a>
      );
    },
  },
  {
    key: "sql-editor",
    type: "render",
    render() {
      const link = router.resolve({
        name: isSQLEditorRoute(router)
          ? WORKSPACE_ROUTE_LANDING
          : SQL_EDITOR_HOME_MODULE,
      });
      return (
        <a
          class="menu-item"
          href={link.fullPath}
          target="_blank"
          rel="noopener noreferrer"
        >
          {isSQLEditorRoute(router)
            ? t(
                "settings.general.workspace.default-landing-page.go-to-workspace"
              )
            : t(
                "settings.general.workspace.default-landing-page.go-to-sql-editor"
              )}
        </a>
      );
    },
  },
  {
    key: "header-divider",
    type: "divider",
  },
  {
    key: "version",
    type: "render",
    render() {
      return <Version tooltipProps={{ placement: "left" }} />;
    },
  },
  {
    key: "debug",
    type: "render",
    show: allowToggleDebug.value,
    render() {
      return (
        <div class="menu-item">
          <div class="flex flex-row items-center gap-x-2 justify-between">
            <span>Debug</span>
            <NSwitch
              size="small"
              value={isDebug.value}
              onUpdate:value={switchDebug}
            />
          </div>
        </div>
      );
    },
  },
  {
    key: "enable-new-layout",
    type: "render",
    render() {
      return (
        <div class="menu-item">
          <div class="flex flex-row items-center gap-x-2 justify-between">
            <span>{t("issue.new-layout")}</span>
            <NSwitch
              size="small"
              value={enabledNewLayout.value}
              onUpdate:value={toggleLayout}
            />
          </div>
        </div>
      );
    },
  },
  {
    key: "header-divider",
    type: "divider",
  },
  {
    key: "logout",
    type: "render",
    render() {
      return (
        <div class="menu-item" onClick={logout}>
          {t("common.logout")}
        </div>
      );
    },
  },
]);
</script>
