import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import {
  PROJECT_V1_ROUTE_DATABASES,
  useCurrentRoute,
  useNavigate,
} from "@/react/router";
import { useAppStore } from "@/react/stores/app";
import {
  bindingMatchesUser,
  getBindingExpirationDate,
  readJson,
  writeJson,
} from "@/react/stores/app/utils";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { displayRoleTitle, formatAbsoluteDateTime } from "@/utils";
import { storageKeyIamRemind } from "@/utils/storage-keys";

interface IAMRemindDialogProps {
  project: Project;
}

export function IAMRemindDialog({ project }: IAMRemindDialogProps) {
  const { t } = useTranslation();
  const route = useCurrentRoute();
  const navigate = useNavigate();
  const currentUser = useAppStore((state) => state.currentUser);
  const roles = useAppStore((state) => state.roles);
  const policy = useAppStore(
    (state) => state.projectPoliciesByName[project.name]
  );
  const [checked, setChecked] = useState(false);
  const [open, setOpen] = useState(false);

  const pendingExpireRoles = useMemo(() => {
    if (!policy || !currentUser) return [];
    const roleByName = new Map(roles.map((role) => [role.name, role]));
    const now = Date.now();
    const expiresBefore = now + 2 * 24 * 60 * 60 * 1000;
    return bindingMatchesUser(policy, currentUser)
      .map((binding) => {
        const role = roleByName.get(binding.role);
        const expiration = getBindingExpirationDate(binding);
        return role && expiration ? { role, expiration } : undefined;
      })
      .filter((item): item is NonNullable<typeof item> => Boolean(item))
      .filter(({ expiration }) => {
        const time = expiration.getTime();
        return time > now && time < expiresBefore;
      })
      .sort((a, b) => a.expiration.getTime() - b.expiration.getTime());
  }, [currentUser, policy, roles]);

  const storageKey = currentUser?.email
    ? storageKeyIamRemind(currentUser.email)
    : "";
  const remindKey =
    pendingExpireRoles.length > 0
      ? `${project.name}.${pendingExpireRoles
          .map(({ role }) => role.name)
          .join("&")}`
      : "";
  const isInProjectPage = route.fullPath.startsWith(`/${project.name}`);

  useEffect(() => {
    if (!storageKey || !remindKey) {
      setOpen(false);
      return;
    }
    const state = readJson<Record<string, boolean>>(storageKey, {});
    if (!(remindKey in state)) {
      writeJson(storageKey, { ...state, [remindKey]: true });
      setOpen(true);
      return;
    }
    setOpen(state[remindKey] ?? false);
  }, [remindKey, storageKey]);

  const close = () => {
    if (checked && storageKey && remindKey) {
      const state = readJson<Record<string, boolean>>(storageKey, {});
      writeJson(storageKey, { ...state, [remindKey]: false });
    }
    setOpen(false);
  };

  if (pendingExpireRoles.length === 0) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="p-6 w-[calc(100vw-10rem)] max-w-[30rem]">
        <DialogTitle>{t("remind.role-expire.title")}</DialogTitle>
        <div className="mt-4 flex flex-col gap-2 text-sm">
          <p>{t("remind.role-expire.content")}</p>
          <ul className="list-disc text-control-light ml-4">
            {pendingExpireRoles.map(({ role, expiration }) => (
              <li key={role.name}>
                {displayRoleTitle(role.name)}:{" "}
                <span className="text-error">
                  {formatAbsoluteDateTime(expiration.getTime())}
                </span>
              </li>
            ))}
          </ul>
        </div>
        <label className="mt-6 flex items-center gap-x-2 text-sm text-control-light">
          <Checkbox
            checked={checked}
            onCheckedChange={(checked) => setChecked(checked)}
          />
          {t("remind.role-expire.checkbox")}
        </label>

        <div className="mt-7 flex justify-end gap-x-2">
          <Button variant="outline" onClick={close}>
            {t("common.dismiss")}
          </Button>
          {!isInProjectPage && (
            <Button
              onClick={() => {
                setOpen(false);
                void navigate.push({
                  name: PROJECT_V1_ROUTE_DATABASES,
                  params: { projectId: project.name.slice("projects/".length) },
                });
              }}
            >
              {t("remind.role-expire.go-to-project")}
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
