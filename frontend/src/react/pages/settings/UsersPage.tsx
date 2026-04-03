import { Plus, Search, Settings } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/react/components/ui/tabs";
import { useVueState } from "@/react/hooks/useVueState";
import { useActuatorV1Store, useSubscriptionV1Store } from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

type TabValue = "USERS" | "MEMBERS" | "GROUPS";

function getInitialTab(isSaaSMode: boolean): TabValue {
  const hash = window.location.hash.replace("#", "").toUpperCase();
  if (hash === "USERS" && !isSaaSMode) return "USERS";
  if (hash === "MEMBERS" && isSaaSMode) return "MEMBERS";
  if (hash === "GROUPS") return "GROUPS";
  return isSaaSMode ? "MEMBERS" : "USERS";
}

export function UsersPage() {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const subscriptionStore = useSubscriptionV1Store();

  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
  const activeUserCount = useVueState(() => actuatorStore.activeUserCount);
  const userCountLimit = useVueState(() => subscriptionStore.userCountLimit);

  const [tab, setTab] = useState<TabValue>(() => getInitialTab(isSaaSMode));
  const [userSearchText, setUserSearchText] = useState("");
  const [groupSearchText, setGroupSearchText] = useState("");

  // Drawer visibility (placeholders for future tasks)
  const [_showCreateUserDrawer, _setShowCreateUserDrawer] = useState(false);
  const [_showCreateGroupDrawer, _setShowCreateGroupDrawer] = useState(false);
  const [_showAadSyncDrawer, _setShowAadSyncDrawer] = useState(false);

  const remainingUserCount = useMemo(
    () => Math.max(0, userCountLimit - activeUserCount),
    [userCountLimit, activeUserCount]
  );

  // Sync tab to URL hash
  useEffect(() => {
    window.location.hash = tab;
  }, [tab]);

  const handleTabChange = (value: TabValue) => {
    if (value) setTab(value);
  };

  return (
    <div className="w-full px-4 overflow-x-hidden flex flex-col pt-2 pb-4">
      {!isSaaSMode && remainingUserCount <= 3 && (
        <Alert variant="warning" className="mb-2">
          <AlertTitle>
            {t("subscription.usage.user-count.title")}
          </AlertTitle>
          <AlertDescription>
            {t("subscription.usage.user-count.description", {
              total: userCountLimit,
              remaining: remainingUserCount,
            })}{" "}
            {t("subscription.usage.upgrade-prompt")}
          </AlertDescription>
        </Alert>
      )}

      <Tabs
        value={tab}
        onValueChange={(val) => handleTabChange(val as TabValue)}
      >
        <div className="flex items-center justify-between">
          <TabsList>
            {!isSaaSMode && (
              <TabsTrigger value="USERS">
                {t("common.users")}
                <span className="ml-1 font-normal text-control-light">
                  ({activeUserCount})
                </span>
              </TabsTrigger>
            )}
            {isSaaSMode && (
              <TabsTrigger value="MEMBERS">
                {t("common.members", { count: 2 })}
              </TabsTrigger>
            )}
            <TabsTrigger value="GROUPS">
              {t("settings.members.groups.self")}
            </TabsTrigger>
          </TabsList>

          <div className="flex items-center gap-x-2">
            {tab === "USERS" && (
              <>
                <div className="relative">
                  <Input
                    placeholder={t("common.filter-by-name")}
                    value={userSearchText}
                    onChange={(e) => setUserSearchText(e.target.value)}
                    className="h-8 text-sm pr-8"
                  />
                  <Search className="absolute right-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-control-placeholder pointer-events-none" />
                </div>
                <Button variant="outline">
                  <Settings className="h-4 w-4 mr-1" />
                  <FeatureBadge
                    feature={PlanFeature.FEATURE_DIRECTORY_SYNC}
                    clickable={false}
                  />
                  {t("settings.members.entra-sync.self")}
                </Button>
                <Button>
                  <Plus className="h-4 w-4 mr-1" />
                  {t("settings.members.add-user")}
                </Button>
              </>
            )}
            {tab === "GROUPS" && (
              <>
                <div className="relative">
                  <Input
                    placeholder={t("common.filter-by-name")}
                    value={groupSearchText}
                    onChange={(e) => setGroupSearchText(e.target.value)}
                    className="h-8 text-sm pr-8"
                  />
                  <Search className="absolute right-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-control-placeholder pointer-events-none" />
                </div>
                <Button variant="outline">
                  <Settings className="h-4 w-4 mr-1" />
                  <FeatureBadge
                    feature={PlanFeature.FEATURE_DIRECTORY_SYNC}
                    clickable={false}
                  />
                  {t("settings.members.entra-sync.self")}
                </Button>
                <Button>
                  <Plus className="h-4 w-4 mr-1" />
                  {t("settings.members.groups.add-group")}
                </Button>
              </>
            )}
            {tab === "MEMBERS" && (
              <>{/* Placeholder for members tab actions */}</>
            )}
          </div>
        </div>

        {!isSaaSMode && (
          <TabsPanel value="USERS">
            <div className="py-4 text-control-light">
              Users content here
            </div>
          </TabsPanel>
        )}
        {isSaaSMode && (
          <TabsPanel value="MEMBERS">
            <div className="py-4 text-control-light">
              Members content here
            </div>
          </TabsPanel>
        )}
        <TabsPanel value="GROUPS">
          <div className="py-4 text-control-light">
            Groups content here
          </div>
        </TabsPanel>
      </Tabs>
    </div>
  );
}
