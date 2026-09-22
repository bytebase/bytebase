import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import type { ConnectError } from "@connectrpc/connect";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { FormField, FormTitle } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";

interface EditUserSheetProps {
  open: boolean;
  user: User | undefined;
  onClose: () => void;
  onUpdated: (user: User) => void;
}

/**
 * Edit the two fields on a user record that are ordinary profile data.
 *
 * Email, password, two-factor and account state are deliberately absent: each
 * carries a consequence of its own and confirms separately from the row menu.
 */
export function EditUserSheet({
  open,
  user,
  onClose,
  onUpdated,
}: EditUserSheetProps) {
  // Freeze the entity while closing so the form stays stable through the
  // sheet's exit animation.
  const openEntityRef = useRef(user);
  if (open) {
    openEntityRef.current = user;
  }
  const stableUser = openEntityRef.current;

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="narrow">
        <EditUserForm
          key={stableUser?.name ?? "none"}
          user={stableUser}
          onClose={onClose}
          onUpdated={onUpdated}
        />
      </SheetContent>
    </Sheet>
  );
}

function EditUserForm({
  user,
  onClose,
  onUpdated,
}: Omit<EditUserSheetProps, "open">) {
  const { t } = useTranslation();
  const updateUser = useAppStore((state) => state.updateUser);

  const initialTitle = user?.title ?? "";
  const initialPhone = user?.phone ?? "";
  const [title, setTitle] = useState(initialTitle);
  const [phone, setPhone] = useState(initialPhone);
  const [saving, setSaving] = useState(false);

  const isDirty = title !== initialTitle || phone !== initialPhone;

  const handleSubmit = async () => {
    if (!user) return;
    const paths: string[] = [];
    if (title !== initialTitle) paths.push("title");
    if (phone !== initialPhone) paths.push("phone");
    if (paths.length === 0) return;

    setSaving(true);
    try {
      const updated = await updateUser(
        create(UpdateUserRequestSchema, {
          user: { ...user, title, phone },
          updateMask: create(FieldMaskSchema, { paths }),
        })
      );
      onUpdated(updated);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      onClose();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: (error as ConnectError).message || t("common.error"),
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("settings.members.admin.edit-user")}</SheetTitle>
      </SheetHeader>

      <SheetBody>
        <div className="flex flex-col gap-y-6">
          {user?.profile?.source && (
            <Alert
              title={t("settings.members.admin.external-source", {
                source: user.profile.source,
              })}
            />
          )}

          <FormField>
            <FormTitle id="edit-user-name-title">{t("common.name")}</FormTitle>
            <Input
              id="edit-user-name"
              aria-labelledby="edit-user-name-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              maxLength={200}
            />
          </FormField>

          <FormField>
            <FormTitle id="edit-user-phone-title">
              {t("settings.profile.phone")}
            </FormTitle>
            <span className="text-sm text-control-placeholder">
              {t("settings.profile.phone-tips")}
            </span>
            <Input
              id="edit-user-phone"
              aria-labelledby="edit-user-phone-title"
              type="tel"
              value={phone}
              autoComplete="off"
              onChange={(e) => setPhone(e.target.value)}
            />
          </FormField>

          <Alert title={t("settings.members.admin.email-immutable-tip")} />
        </div>
      </SheetBody>

      <SheetFooter>
        <Button appearance="outline" onClick={onClose}>
          {t("common.cancel")}
        </Button>
        <Button disabled={!isDirty || saving} onClick={handleSubmit}>
          {t("common.update")}
        </Button>
      </SheetFooter>
    </>
  );
}
