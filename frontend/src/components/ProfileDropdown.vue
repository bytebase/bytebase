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

<script lang="ts" setup>
import type { DropdownOption } from "naive-ui";
import { NSwitch, NDropdown } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, ref, h } from "vue";
import { useI18n } from "vue-i18n";
import { useLanguage } from "@/composables/useLanguage";
import {
  useActiveUsers,
  useActuatorV1Store,
  useAppFeature,
  useAuthStore,
  useCurrentUserV1,
  useSubscriptionV1Store,
  useUIStateStore,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV2, isDev } from "@/utils";
import ProfilePreview from "./ProfilePreview.vue";
import UserAvatar from "./User/UserAvatar.vue";
import Version from "./misc/Version.vue";

const { t } = useI18n();

const props = defineProps<{
  link?: boolean;
}>();

const actuatorStore = useActuatorV1Store();
const authStore = useAuthStore();
const subscriptionStore = useSubscriptionV1Store();
const uiStateStore = useUIStateStore();
const { setLocale, locale } = useLanguage();
const currentUserV1 = useCurrentUserV1();
const showDropdown = ref(false);
const hideQuickstart = computed(() => {
  if (useAppFeature("bb.feature.hide-quick-start").value) {
    return true;
  }
  // Hide quickstart if there are more than 1 active users.
  return useActiveUsers().length > 1;
});
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
  showDropdown.value = false;
};

const { isDebug } = storeToRefs(actuatorStore);

const switchDebug = () => {
  actuatorStore.patchDebug({
    debug: !isDebug.value,
  });
};

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
        return h(
          "div",
          {
            key: item.value,
            class: classes.join(" "),
            onClick: () => toggleLocale(item.value),
          },
          h("div", { class: "radio cursor-pointer text-sm" }, [
            h("input", {
              type: "radio",
              class: "btn",
              checked: locale.value === item.value,
            }),
            h("label", { class: "ml-2 cursor-pointer" }, item.label),
          ])
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
        return h(
          "div",
          {
            key: item.value,
            class: classes.join(" "),
            onClick: () => switchPlan(item.value),
          },
          h("div", { class: "radio cursor-pointer text-sm" }, [
            h("input", {
              type: "radio",
              class: "btn",
              checked: currentPlan.value === item.plan,
            }),
            h("label", { class: "ml-2 cursor-pointer" }, item.label),
          ])
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
      return h(ProfilePreview, {
        link: props.link,
        onClick: () => (showDropdown.value = false),
      });
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
      return h(
        NDropdown,
        {
          key: "switch-language",
          options: languageOptions.value,
          placement: "left-start",
        },
        () =>
          h(
            "div",
            {
              class: "menu-item",
            },
            t("common.language")
          )
      );
    },
  },
  {
    key: "license",
    type: "render",
    show: isDev(),
    render() {
      return h(
        NDropdown,
        {
          key: "switch-license",
          options: licenseOptions.value,
          placement: "left-start",
        },
        () =>
          h(
            "div",
            {
              class: "menu-item",
            },
            t("common.license")
          )
      );
    },
  },
  {
    key: "quick-start",
    type: "render",
    show: !hideQuickstart.value,
    render() {
      return h(
        "div",
        {
          class: "menu-item",
          onClick: resetQuickstart,
        },
        t("quick-start.self")
      );
    },
  },
  {
    key: "help",
    type: "render",
    show: !hideHelp.value,
    render() {
      return h(
        "a",
        {
          class: "menu-item",
          target: "_blank",
          href: "https://bytebase.com/docs?source=console",
        },
        t("common.help")
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
      return h(Version, { tooltipProps: { placement: "left" } });
    },
  },
  {
    key: "debug",
    type: "render",
    show: allowToggleDebug.value,
    render() {
      return h(
        "div",
        {
          class: "menu-item",
        },
        [
          h(
            "div",
            {
              class: "flex flex-row items-center space-x-2 justify-between",
            },
            [
              h("span", {}, "Debug"),
              h(NSwitch, {
                size: "small",
                value: isDebug.value,
                "onUpdate:value": switchDebug,
              }),
            ]
          ),
        ]
      );
    },
  },
  {
    key: "header-divider",
    type: "divider",
    show: allowToggleDebug.value,
  },
  {
    key: "logout",
    type: "render",
    render() {
      return h(
        "div",
        {
          class: "menu-item",
          onClick: logout,
        },
        t("common.logout")
      );
    },
  },
]);
</script>
