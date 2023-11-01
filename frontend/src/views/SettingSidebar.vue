<template>
  <CommonSidebar
    type="route"
    :show-go-back="true"
    :item-list="(settingSidebarItemList as SidebarItem[])"
    :get-item-class="getItemClass"
  />
</template>

<script lang="ts" setup>
import {
  UserCircle,
  Building,
  ShieldCheck,
  Link,
  Archive,
} from "lucide-vue-next";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { SidebarItem } from "@/components/CommonSidebar.vue";
import { useCurrentUserV1 } from "@/store";
import { hasWorkspacePermissionV1 } from "../utils";

const { t } = useI18n();
const route = useRoute();
const currentUserV1 = useCurrentUserV1();

const getItemClass = (path: string | undefined) => {
  const list = [];
  if (route.name === "workspace.profile") {
    if (path === "/setting/member") {
      list.push("router-link-active", "bg-link-hover");
    }
  }
  return list;
};

const showProjectItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-project",
    currentUserV1.value.userRole
  );
});

const showSensitiveDataItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});

const showAccessControlItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-access-control",
    currentUserV1.value.userRole
  );
});

const showSSOItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sso",
    currentUserV1.value.userRole
  );
});

const showVCSItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-vcs-provider",
    currentUserV1.value.userRole
  );
});

const showIntegrationSection = computed(() => {
  return showVCSItem.value || showSSOItem.value;
});

const showDebugLogItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.debug-log",
    currentUserV1.value.userRole
  );
});

const showAuditLogItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.audit-log",
    currentUserV1.value.userRole
  );
});

const showMailDeliveryItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-mail-delivery",
    currentUserV1.value.userRole
  );
});

const settingSidebarItemList = computed((): SidebarItem[] => {
  return [
    {
      title: t("settings.sidebar.account"),
      icon: h(UserCircle),
      children: [
        {
          title: t("settings.sidebar.profile"),
          path: "/setting/profile",
        },
      ],
    },
    {
      title: t("settings.sidebar.workspace"),
      icon: h(Building),
      children: [
        {
          title: t("settings.sidebar.general"),
          path: "/setting/general",
        },
        {
          title: t("settings.sidebar.members"),
          path: "/setting/member",
        },
        {
          title: t("settings.sidebar.custom-roles"),
          path: "/setting/role",
        },
        {
          title: t("common.projects"),
          path: "/setting/project",
          hide: !showProjectItem.value,
        },
        {
          title: t("settings.sidebar.subscription"),
          path: "/setting/subscription",
        },
        {
          title: t("settings.sidebar.debug-log"),
          path: "/setting/debug-log",
          hide: !showDebugLogItem.value,
        },
      ],
    },
    {
      title: t("settings.sidebar.security-and-policy"),
      icon: h(ShieldCheck),
      children: [
        {
          title: t("sql-review.title"),
          path: "/setting/sql-review",
        },
        {
          title: t("slow-query.self"),
          path: "/setting/slow-query",
        },
        {
          title: t("schema-template.self"),
          path: "/setting/schema-template",
        },
        {
          title: t("custom-approval.risk.risk-center"),
          path: "/setting/risk-center",
        },
        {
          title: t("custom-approval.self"),
          path: "/setting/custom-approval",
        },
        {
          title: t("settings.sidebar.sensitive-data"),
          path: "/setting/sensitive-data",
          hide: !showSensitiveDataItem.value,
        },
        {
          title: t("settings.sidebar.access-control"),
          path: "/setting/access-control",
          hide: !showAccessControlItem.value,
        },
        {
          title: t("settings.sidebar.audit-log"),
          path: "/setting/audit-log",
          hide: !showAuditLogItem.value,
        },
      ],
    },
    {
      title: t("settings.sidebar.integration"),
      icon: h(Link),
      hide: !showIntegrationSection.value,
      children: [
        {
          title: t("settings.sidebar.gitops"),
          path: "/setting/gitops",
        },
        {
          title: t("settings.sidebar.sso"),
          hide: !showSSOItem.value,
          path: "/setting/sso",
        },
        {
          title: t("settings.sidebar.mail-delivery"),
          hide: !showMailDeliveryItem.value,
          path: "/setting/mail-delivery",
        },
      ],
    },
    {
      title: t("common.archived"),
      icon: h(Archive),
      path: "/archive",
    },
  ];
});
</script>
