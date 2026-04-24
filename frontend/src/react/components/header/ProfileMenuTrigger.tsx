import { ChevronRight } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { UserAvatar } from "@/react/components/UserAvatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSubmenu,
  DropdownMenuSubmenuContent,
  DropdownMenuSubmenuTrigger,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import {
  useAppFeature,
  useCurrentUser,
  useQuickstartReset,
  useServerInfo,
  useSubscription,
  useWorkspace,
} from "@/react/hooks/useAppState";
import {
  AUTH_SIGNIN_MODULE,
  isSqlEditorRouteName,
  SETTING_ROUTE_PROFILE,
  SQL_EDITOR_HOME_MODULE,
  useCurrentRoute,
  useNavigate,
  WORKSPACE_ROUTE_LANDING,
} from "@/react/router";
import { useAppStore } from "@/react/stores/app";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { isDev } from "@/utils/util";
import { HEADER_LANGUAGE_OPTIONS, setAppLocale } from "./common";
import { VersionMenuItem } from "./VersionMenuItem";

export interface ProfileMenuProps {
  size?: "small" | "medium";
  link?: boolean;
}

export function ProfileMenuTrigger({
  size = "medium",
  link = true,
}: ProfileMenuProps) {
  const { t, i18n } = useTranslation();
  const currentUser = useCurrentUser();
  const serverInfo = useServerInfo();
  const { subscription, uploadLicense } = useSubscription();
  const workspace = useWorkspace();
  const route = useCurrentRoute();
  const navigate = useNavigate();
  const resetQuickstartProgress = useQuickstartReset();
  const hideQuickStart = useAppFeature("bb.feature.hide-quick-start");
  const currentPlan = subscription?.plan ?? PlanType.FREE;
  const quickStartEnabled =
    !hideQuickStart &&
    Boolean(serverInfo?.enableSample) &&
    (serverInfo?.activatedUserCount ?? 0) <= 1;
  const customLogo = workspace?.logo ?? "";
  const [open, setOpen] = useState(false);

  const wrapperClass = useMemo(() => {
    if (!customLogo) {
      return "flex items-center justify-center rounded-3xl bg-gray-100";
    }
    return size === "small"
      ? "flex items-center justify-center rounded-3xl bg-gray-100 md:px-1 md:py-0.5"
      : "flex items-center justify-center rounded-3xl bg-gray-100 md:px-2 md:py-1.5";
  }, [customLogo, size]);

  const logoClass = size === "small" ? "mr-2" : "mr-4";

  const sqlEditorMenuLabel = isSqlEditorRouteName(route.name)
    ? t("settings.general.workspace.default-landing-page.go-to-workspace")
    : t("settings.general.workspace.default-landing-page.go-to-sql-editor");

  const handleProfileNavigate = () => {
    if (!link) return;
    setOpen(false);
    void navigate.push({ name: SETTING_ROUTE_PROFILE });
  };

  const handleWorkspaceToggle = () => {
    const target = navigate.resolve({
      name: isSqlEditorRouteName(route.name)
        ? WORKSPACE_ROUTE_LANDING
        : SQL_EDITOR_HOME_MODULE,
    });
    setOpen(false);
    window.open(target.fullPath, "_blank", "noopener,noreferrer");
  };

  const switchPlan = (license: string) => {
    void uploadLicense(license);
    setOpen(false);
  };

  return (
    <div className={wrapperClass}>
      {customLogo ? (
        <img
          src={customLogo}
          alt={t("settings.general.workspace.logo")}
          className={`ml-1 hidden h-6 bg-center bg-no-repeat object-contain md:block ${logoClass}`}
        />
      ) : null}

      <DropdownMenu open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger
          render={
            <button type="button" className="cursor-pointer rounded-full" />
          }
        >
          <UserAvatar
            size="sm"
            className="cursor-pointer"
            title={currentUser?.title || currentUser?.email || ""}
          />
        </DropdownMenuTrigger>

        <DropdownMenuContent className="w-56 p-0">
          <DropdownMenuItem
            className="block w-full px-4 py-3"
            onClick={handleProfileNavigate}
          >
            <div className="text-left">
              <p className="flex justify-between gap-x-2 text-sm">
                <span className="truncate font-medium text-main">
                  {currentUser?.title}
                </span>
              </p>
              <p className="truncate text-sm text-control">
                {currentUser?.email}
              </p>
            </div>
          </DropdownMenuItem>

          <DropdownMenuSeparator className="mx-0" />

          <DropdownMenuSubmenu>
            <DropdownMenuSubmenuTrigger className="justify-between">
              {t("common.language")}
              <ChevronRight className="h-4 w-4 text-control-light" />
            </DropdownMenuSubmenuTrigger>
            <DropdownMenuSubmenuContent className="w-48">
              {HEADER_LANGUAGE_OPTIONS.map((item) => (
                <DropdownMenuItem
                  key={item.value}
                  className={
                    item.value === i18n.language ? "bg-control-bg" : ""
                  }
                  onClick={() => {
                    setAppLocale(item.value);
                    setOpen(false);
                  }}
                >
                  <span className="mr-2">
                    <input
                      type="radio"
                      readOnly
                      checked={item.value === i18n.language}
                    />
                  </span>
                  {item.label}
                </DropdownMenuItem>
              ))}
            </DropdownMenuSubmenuContent>
          </DropdownMenuSubmenu>

          {isDev() ? (
            <DropdownMenuSubmenu>
              <DropdownMenuSubmenuTrigger className="justify-between">
                {t("common.license")}
                <ChevronRight className="h-4 w-4 text-control-light" />
              </DropdownMenuSubmenuTrigger>
              <DropdownMenuSubmenuContent className="w-48">
                {[
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
                ].map((item) => (
                  <DropdownMenuItem
                    key={item.plan}
                    className={item.plan === currentPlan ? "bg-control-bg" : ""}
                    onClick={() => switchPlan(item.value)}
                  >
                    <span className="mr-2">
                      <input
                        type="radio"
                        readOnly
                        checked={item.plan === currentPlan}
                      />
                    </span>
                    {item.label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuSubmenuContent>
            </DropdownMenuSubmenu>
          ) : null}

          {quickStartEnabled ? (
            <DropdownMenuItem
              onClick={() => {
                resetQuickstartProgress();
                setOpen(false);
              }}
            >
              {t("quick-start.self")}
            </DropdownMenuItem>
          ) : null}

          <DropdownMenuItem onClick={handleWorkspaceToggle}>
            {sqlEditorMenuLabel}
          </DropdownMenuItem>

          <DropdownMenuSeparator className="mx-0" />

          <VersionMenuItem onCloseMenu={() => setOpen(false)} />

          <DropdownMenuSeparator className="mx-0" />

          <DropdownMenuItem
            onClick={() => {
              setOpen(false);
              const redirect =
                route.fullPath.startsWith("/auth") || !route.fullPath
                  ? undefined
                  : route.fullPath;
              const signinUrl = navigate.resolve({
                name: AUTH_SIGNIN_MODULE,
                query: { redirect },
              }).fullPath;
              void useAppStore.getState().logout(signinUrl);
            }}
          >
            {t("common.logout")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
