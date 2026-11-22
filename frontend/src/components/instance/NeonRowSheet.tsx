import type { TFunction } from "i18next";
import { useId, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { SecretInput } from "@/components/SecretInput";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  FormError,
  FormField,
  FormFieldGroup,
  FormLabel,
  ResponsiveFormLayout,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Switch } from "@/components/ui/switch";
import { useEnvironmentList } from "@/hooks/useAppState";
import { formatEnvironmentName } from "@/types/v1/environment";
import { type NeonOption, NeonOptionSelect } from "./NeonOptionSelect";
import {
  branchKey,
  effectiveSettings,
  hasOverride,
  isValidInstanceId,
  NEON_SYNC_INTERVALS,
  type NeonImportDraft,
  type NeonScheduleMode,
  type NeonSharedSettings,
  syncIntervalLabel,
  withDatabaseToggled,
  withRole,
} from "./neon-import";

export const SHARED = "__shared__";
const NONE = "__none__";

export function intervalOptions(t: TFunction): NeonOption[] {
  return NEON_SYNC_INTERVALS.map((seconds) => ({
    value: String(seconds),
    label: syncIntervalLabel(t, seconds),
  }));
}

// The schedule choices of one row: the shared value while the row inherits,
// then every interval.
export function rowIntervalOptions(
  t: TFunction,
  shared?: NeonSharedSettings
): NeonOption[] {
  return [
    ...(shared
      ? [
          {
            value: SHARED,
            label: t("instance.neon.configure.shared-option", {
              value: syncIntervalLabel(t, shared.syncSeconds),
            }),
          },
        ]
      : []),
    ...intervalOptions(t),
  ];
}

export const rowIntervalFromValue = (value: string): number | undefined =>
  value === SHARED ? undefined : Number(value);

export function useEnvironmentTitle() {
  const environments = useEnvironmentList();
  const { t } = useTranslation();
  return useMemo(() => {
    const titles = new Map(
      environments.map((environment) => [
        formatEnvironmentName(environment.id),
        environment.title,
      ])
    );
    return (name: string) =>
      name
        ? (titles.get(name) ?? name)
        : t("instance.neon.configure.no-environment");
  }, [environments, t]);
}

// The environment choices of one row: the shared value while the row inherits,
// every environment, and an explicit "no environment".
export function useRowEnvironmentOptions(shared?: NeonSharedSettings) {
  const { t } = useTranslation();
  const environments = useEnvironmentList();
  const environmentTitle = useEnvironmentTitle();
  return useMemo(
    () => [
      ...(shared
        ? [
            {
              value: SHARED,
              label: t("instance.neon.configure.shared-option", {
                value: environmentTitle(shared.environment),
              }),
            },
          ]
        : []),
      ...environments.map((environment) => ({
        value: formatEnvironmentName(environment.id),
        label: environment.title,
      })),
      { value: NONE, label: t("instance.neon.configure.no-environment") },
    ],
    [environments, environmentTitle, shared, t]
  );
}

export const rowEnvironmentValue = (
  draft: NeonImportDraft,
  inherits: boolean
): string =>
  draft.environment === undefined
    ? inherits
      ? SHARED
      : NONE
    : draft.environment || NONE;

export const rowEnvironmentFromValue = (value: string): string | undefined =>
  value === SHARED ? undefined : value === NONE ? "" : value;

export function NeonRowSheet({
  open,
  draft,
  shared,
  duplicateIds,
  schedule,
  onChange,
  onClose,
}: {
  open: boolean;
  draft?: NeonImportDraft;
  // Absent once settings are frozen for import, when a row no longer inherits.
  shared?: NeonSharedSettings;
  duplicateIds: ReadonlySet<string>;
  schedule: NeonScheduleMode;
  onChange: (update: (draft: NeonImportDraft) => NeonImportDraft) => void;
  onClose: () => void;
}) {
  // Keep the last row in place while the sheet animates closed.
  const openDraftRef = useRef(draft);
  if (open && draft) openDraftRef.current = draft;
  const stable = openDraftRef.current;

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        {stable && (
          <RowForm
            key={branchKey(stable.branch)}
            draft={stable}
            shared={shared}
            duplicateIds={duplicateIds}
            schedule={schedule}
            onChange={onChange}
            onClose={onClose}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}

function RowForm({
  draft,
  shared,
  duplicateIds,
  schedule,
  onChange,
  onClose,
}: {
  draft: NeonImportDraft;
  shared?: NeonSharedSettings;
  duplicateIds: ReadonlySet<string>;
  schedule: NeonScheduleMode;
  onChange: (update: (draft: NeonImportDraft) => NeonImportDraft) => void;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const id = useId();
  const { branch } = draft;
  const effective = effectiveSettings(
    draft,
    shared ?? { environment: "", syncAll: true, syncSeconds: 0 }
  );
  const environmentOptions = useRowEnvironmentOptions(shared);
  const idError = !isValidInstanceId(draft.instanceId)
    ? t("resource-id.validation.pattern", { resource: t("common.instance") })
    : duplicateIds.has(draft.instanceId)
      ? t("instance.neon.sheet.id-duplicate")
      : "";
  const titleError = draft.title.trim()
    ? ""
    : t("instance.neon.sheet.title-required");
  const passwordMissing = draft.passwordSource === "typed" && !draft.password;

  return (
    <form
      className="flex min-h-0 flex-1 flex-col"
      onSubmit={(event) => {
        event.preventDefault();
        onClose();
      }}
    >
      <SheetHeader>
        <SheetTitle className="truncate">
          {draft.title || branch.branchTitle}
        </SheetTitle>
        <SheetDescription className="break-words">
          {t("instance.neon.sheet.description", {
            project: branch.projectTitle || branch.projectId,
            projectId: branch.projectId,
            branchId: branch.branchId,
          })}
        </SheetDescription>
      </SheetHeader>
      <SheetBody>
        <ResponsiveFormLayout
          layout="vertical"
          density="compact"
          className="flex flex-col gap-6"
        >
          <section className="flex flex-col gap-4">
            <h3 className="text-sm font-semibold text-main">
              {t("instance.neon.sheet.identity")}
            </h3>
            <FormFieldGroup density="compact">
              <FormField
                title={
                  <FormLabel htmlFor={`${id}-title`}>
                    {t("instance.instance-name")}
                  </FormLabel>
                }
              >
                <Input
                  id={`${id}-title`}
                  value={draft.title}
                  maxLength={200}
                  autoComplete="off"
                  aria-invalid={!!titleError || undefined}
                  aria-describedby={
                    titleError ? `${id}-title-error` : undefined
                  }
                  onChange={(event) => {
                    const title = event.target.value;
                    onChange((row) => ({ ...row, title }));
                  }}
                />
                {titleError && (
                  <FormError id={`${id}-title-error`}>{titleError}</FormError>
                )}
              </FormField>
              <FormField
                title={
                  <FormLabel htmlFor={`${id}-id`}>
                    {t("instance.neon.sheet.instance-id")}
                  </FormLabel>
                }
                description={t("instance.neon.sheet.instance-id-help")}
              >
                <Input
                  id={`${id}-id`}
                  value={draft.instanceId}
                  className="font-mono"
                  autoComplete="off"
                  spellCheck={false}
                  aria-invalid={!!idError || undefined}
                  aria-describedby={idError ? `${id}-id-error` : undefined}
                  onChange={(event) => {
                    const instanceId = event.target.value.trim();
                    onChange((row) => ({ ...row, instanceId }));
                  }}
                />
                {idError && (
                  <FormError id={`${id}-id-error`}>{idError}</FormError>
                )}
              </FormField>
              <FormField
                title={
                  <FormLabel htmlFor={`${id}-environment`}>
                    {t("common.environment")}
                  </FormLabel>
                }
              >
                <NeonOptionSelect
                  id={`${id}-environment`}
                  value={rowEnvironmentValue(draft, !!shared)}
                  options={environmentOptions}
                  onChange={(value) =>
                    onChange((row) => ({
                      ...row,
                      environment: rowEnvironmentFromValue(value),
                    }))
                  }
                />
              </FormField>
            </FormFieldGroup>
          </section>

          <Separator />

          <section className="flex flex-col gap-4">
            <h3 className="text-sm font-semibold text-main">
              {t("instance.neon.sheet.connection")}
            </h3>
            <FormFieldGroup density="compact">
              <FormField
                title={
                  <FormLabel htmlFor={`${id}-role`}>
                    {t("instance.neon.sheet.role")}
                  </FormLabel>
                }
              >
                <NeonOptionSelect
                  id={`${id}-role`}
                  value={draft.role}
                  invalid={!draft.role}
                  placeholder={t("instance.neon.choose-role")}
                  aria-describedby={!draft.role ? `${id}-role-note` : undefined}
                  options={branch.roles.map((role) => ({
                    value: role,
                    label: role,
                  }))}
                  onChange={(role) => onChange((row) => withRole(row, role))}
                />
                {!draft.role && (
                  <FormError id={`${id}-role-note`}>
                    {t("instance.neon.sheet.role-required")}
                  </FormError>
                )}
              </FormField>
              <FormField
                title={
                  <FormLabel htmlFor={`${id}-password-source`}>
                    {t("common.password")}
                  </FormLabel>
                }
              >
                <NeonOptionSelect
                  id={`${id}-password-source`}
                  value={draft.passwordSource}
                  options={[
                    {
                      value: "neon",
                      label: t("instance.neon.sheet.password-neon"),
                    },
                    {
                      value: "typed",
                      label: t("instance.neon.sheet.password-typed"),
                    },
                  ]}
                  onChange={(source) =>
                    onChange((row) => ({
                      ...row,
                      passwordSource: source === "typed" ? "typed" : "neon",
                      password: "",
                    }))
                  }
                />
                {draft.passwordSource === "typed" && (
                  <SecretInput
                    isCreating
                    autoComplete="off"
                    value={draft.password}
                    aria-label={t("instance.neon.password-for", {
                      role: draft.role,
                    })}
                    aria-invalid={passwordMissing || undefined}
                    aria-describedby={
                      passwordMissing ? `${id}-password-note` : undefined
                    }
                    placeholder={t("instance.neon.password-for", {
                      role: draft.role,
                    })}
                    onValueChange={(password) =>
                      onChange((row) => ({ ...row, password }))
                    }
                  />
                )}
                {passwordMissing && (
                  <FormError id={`${id}-password-note`}>
                    {t("instance.neon.sheet.password-required")}
                  </FormError>
                )}
              </FormField>
              <FormField
                title={
                  <FormLabel htmlFor={`${id}-database`}>
                    {t("instance.neon.sheet.database")}
                  </FormLabel>
                }
              >
                <NeonOptionSelect
                  id={`${id}-database`}
                  value={draft.database}
                  options={branch.databases.map((database) => ({
                    value: database.name,
                    label: database.name,
                  }))}
                  onChange={(database) =>
                    onChange((row) => ({ ...row, database }))
                  }
                />
              </FormField>
            </FormFieldGroup>
          </section>

          <Separator />

          <section className="flex flex-col gap-4">
            <h3 className="text-sm font-semibold text-main">
              {t("instance.neon.sheet.sync")}
            </h3>
            <FormFieldGroup density="compact">
              <FormField title={t("instance.sync-databases.sync-all")}>
                <div className="flex items-center py-0.5">
                  <Switch
                    aria-label={t("instance.sync-databases.sync-all")}
                    checked={effective.syncAll}
                    onCheckedChange={(syncAll) =>
                      onChange((row) => ({ ...row, syncAll }))
                    }
                  />
                </div>
                {!effective.syncAll && (
                  <fieldset
                    aria-label={t("instance.neon.sheet.databases-legend")}
                    className="flex flex-col gap-1.5 border-l-2 border-block-border pl-3"
                  >
                    {branch.databases.map((database) => (
                      <label
                        key={database.name}
                        className="flex cursor-pointer items-center gap-2 py-0.5 text-sm text-main select-none"
                      >
                        <Checkbox
                          aria-label={database.name}
                          checked={draft.databases.includes(database.name)}
                          onCheckedChange={(checked) =>
                            onChange((row) =>
                              withDatabaseToggled(row, database.name, checked)
                            )
                          }
                        />
                        <span>{database.name}</span>
                        <span className="text-xs text-control-light">
                          {t("instance.neon.select.owner", {
                            owner: database.owner,
                          })}
                        </span>
                      </label>
                    ))}
                  </fieldset>
                )}
                {!effective.syncAll && draft.databases.length === 0 && (
                  <p className="text-xs leading-4 text-control-light">
                    {t("instance.neon.sheet.scope-none")}
                  </p>
                )}
              </FormField>
              {schedule === "editable" && (
                <FormField
                  title={
                    <FormLabel htmlFor={`${id}-interval`}>
                      {t("instance.neon.configure.interval")}
                    </FormLabel>
                  }
                >
                  <NeonOptionSelect
                    id={`${id}-interval`}
                    value={
                      shared && draft.syncSeconds === undefined
                        ? SHARED
                        : String(effective.syncSeconds)
                    }
                    options={rowIntervalOptions(t, shared)}
                    onChange={(value) =>
                      onChange((row) => ({
                        ...row,
                        syncSeconds: rowIntervalFromValue(value),
                      }))
                    }
                  />
                </FormField>
              )}
            </FormFieldGroup>
          </section>
        </ResponsiveFormLayout>
      </SheetBody>
      <SheetFooter>
        {shared && hasOverride(draft) && (
          <Button
            type="button"
            appearance="outline"
            onClick={() =>
              onChange((row) => ({
                ...row,
                environment: undefined,
                syncAll: undefined,
                syncSeconds: undefined,
              }))
            }
          >
            {t("instance.neon.sheet.reset")}
          </Button>
        )}
        <Button type="submit">{t("common.done")}</Button>
      </SheetFooter>
    </form>
  );
}
