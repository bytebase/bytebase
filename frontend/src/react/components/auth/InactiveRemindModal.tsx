import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { useVueState } from "@/react/hooks/useVueState";
import { useAuthStore, useCurrentUserV1, useSettingV1Store } from "@/store";
import { storageKeyLastActivity } from "@/utils/storage-keys";

const SHOW_THRESHOLD_MIN = 3;

export function InactiveRemindModal() {
  const { t } = useTranslation();
  const currentUserEmail = useVueState(() => useCurrentUserV1().value.email);
  const inactiveTimeoutInSeconds = useVueState(() =>
    Number(
      useSettingV1Store().workspaceProfile.inactiveSessionTimeout?.seconds ?? 0
    )
  );

  const storageKey = storageKeyLastActivity(currentUserEmail);
  const [nowMS, setNowMS] = useState(() => Date.now());
  const [lastActivityTs, setLastActivityTs] = useState(() => Date.now());
  const logoutTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    const read = () => {
      const raw = localStorage.getItem(storageKey);
      if (raw !== null) {
        const parsed = Number(raw);
        if (!Number.isNaN(parsed)) setLastActivityTs(parsed);
      }
    };
    read();
    // Poll for same-tab updates (VueUse writes aren't broadcast as `storage`
    // events in the same tab).
    const pollId = setInterval(read, 1000);
    const onStorage = (e: StorageEvent) => {
      if (e.key === storageKey) read();
    };
    window.addEventListener("storage", onStorage);
    return () => {
      clearInterval(pollId);
      window.removeEventListener("storage", onStorage);
    };
  }, [storageKey]);

  useEffect(() => {
    const id = setInterval(() => setNowMS(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);

  const shouldShow = (() => {
    if (inactiveTimeoutInSeconds <= 0) return false;
    const inactiveSeconds = (nowMS - lastActivityTs) / 1000;
    return inactiveSeconds > inactiveTimeoutInSeconds - SHOW_THRESHOLD_MIN * 60;
  })();

  useEffect(() => {
    if (!shouldShow) {
      if (logoutTimerRef.current) {
        clearTimeout(logoutTimerRef.current);
        logoutTimerRef.current = null;
      }
      return;
    }
    logoutTimerRef.current = setTimeout(
      () => {
        useAuthStore().logout();
      },
      SHOW_THRESHOLD_MIN * 60 * 1000
    );
    return () => {
      if (logoutTimerRef.current) {
        clearTimeout(logoutTimerRef.current);
        logoutTimerRef.current = null;
      }
    };
  }, [shouldShow]);

  const staySignedIn = () => {
    const now = Date.now();
    localStorage.setItem(storageKey, String(now));
    setLastActivityTs(now);
  };

  const logout = () => useAuthStore().logout();

  if (!shouldShow) return null;

  return (
    <Dialog open>
      <DialogContent className="p-4 md:min-w-96 max-w-full">
        <DialogTitle className="font-medium text-lg">
          {t("auth.inactive-modal.title")}
        </DialogTitle>
        <DialogDescription className="textinfo mt-1">
          {t("auth.inactive-modal.description", {
            minutes: SHOW_THRESHOLD_MIN,
          })}
        </DialogDescription>
        <div className="w-full flex items-center justify-end gap-x-2 mt-4">
          <Button variant="ghost" onClick={logout}>
            {t("common.logout")}
          </Button>
          <Button onClick={staySignedIn}>
            {t("auth.inactive-modal.stay-signed-in")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
