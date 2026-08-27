import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { MCPMethodClass } from "@/types/proto-es/v1/annotation_pb";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import type { MCPInfo } from "@/types/proto-es/v1/workspace_service_pb";
import {
  MCPEngineEnforcement_Masking,
  MCPEngineEnforcement_ReadOnlyDepth,
} from "@/types/proto-es/v1/workspace_service_pb";
import { engineNameV1 } from "@/utils/v1/instance";
import {
  engineNotes,
  groupEnginesByMasking,
  groupEnginesByReadOnlyDepth,
  groupMethodsByService,
  methodsServedBy,
  modeFor,
} from "./mcpPolicy";

const engineLabel = (engine: Engine): string =>
  engineNameV1(engine) || Engine[engine] || String(engine);

interface Props {
  readonly open: boolean;
  readonly capability: MCPSetting_Capability;
  /**
   * Required, and every caller gates its trigger on having it. An optional one
   * would let a pending or refused GetMCPInfo render as a mode that serves
   * nothing — "0 of 0" over a workspace whose ceiling this drawer exists to
   * explain. GetMCPInfo refuses outright under an invalid or unserved
   * ceiling (BOT-106), which is exactly when an admin opens this.
   */
  readonly info: MCPInfo;
  readonly modeLabel: string;
  /**
   * The masking answer this drawer describes. Required, not defaulted from
   * `info`: the settings form previews a candidate policy the admin has not
   * saved, the consent page describes the one in force, and defaulting to
   * either silently gives the other consumer the wrong one.
   */
  readonly ignoreMaskingExemptions: boolean;
  readonly onClose: () => void;
}

export function MCPModeContentsSheet({
  open,
  capability,
  info,
  modeLabel,
  ignoreMaskingExemptions,
  onClose,
}: Props) {
  const { t } = useTranslation();

  const served = useMemo(
    () => methodsServedBy(modeFor(info, capability), info.methods),
    [info, capability]
  );
  const groups = useMemo(() => groupMethodsByService(served), [served]);
  // What this mode adds over Read-only: the WRITE class is the only one a
  // lower ceiling does not serve, so highlighting it is the mode delta.
  const writeCount = useMemo(
    () => served.filter((m) => m.class === MCPMethodClass.WRITE).length,
    [served]
  );
  const depthGroups = useMemo(
    () => groupEnginesByReadOnlyDepth(info.engines),
    [info]
  );
  const maskingGroups = useMemo(
    () => groupEnginesByMasking(info.engines),
    [info]
  );
  const notes = useMemo(() => engineNotes(info.engines), [info]);

  const depthText = (depth: MCPEngineEnforcement_ReadOnlyDepth): string => {
    switch (depth) {
      case MCPEngineEnforcement_ReadOnlyDepth.STATEMENT_AND_SESSION:
        return t("settings.mcp.contents.depth.statement-and-session");
      case MCPEngineEnforcement_ReadOnlyDepth.STATEMENT:
        return t("settings.mcp.contents.depth.statement");
      case MCPEngineEnforcement_ReadOnlyDepth.UNSUPPORTED:
        return t("settings.mcp.contents.depth.unsupported");
      default:
        return t("settings.mcp.contents.depth.unspecified");
    }
  };

  // The server's note is English prose and is never rendered. The console keys
  // off the engine, which is what keeps the caveat translated — and a newer
  // backend can carry a note for an engine this bundle has no wording for, so
  // the caveat is still announced rather than dropped or shown in English.
  // The caller only passes engines that have a note (engineNotes filters).
  const noteText = (engine: Engine): string =>
    engine === Engine.REDSHIFT
      ? t("settings.mcp.contents.notes.redshift")
      : t("settings.mcp.contents.notes.unavailable");

  const maskingText = (masking: MCPEngineEnforcement_Masking): string => {
    switch (masking) {
      case MCPEngineEnforcement_Masking.COLUMN:
        return t("settings.mcp.contents.masking.column");
      case MCPEngineEnforcement_Masking.DOCUMENT:
        return t("settings.mcp.contents.masking.document");
      case MCPEngineEnforcement_Masking.NONE:
        return t("settings.mcp.contents.masking.none");
      default:
        return t("settings.mcp.contents.masking.unspecified");
    }
  };

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>
            {t("settings.mcp.contents.title", { mode: modeLabel })}
          </SheetTitle>
          <SheetDescription>
            {t("settings.mcp.contents.methods.count", {
              served: served.length,
              total: info?.methods.length ?? 0,
            })}
            {writeCount > 0 && (
              <>
                {" "}
                {t("settings.mcp.contents.methods.write-legend", {
                  count: writeCount,
                })}
              </>
            )}
          </SheetDescription>
        </SheetHeader>

        <SheetBody>
          <div className="flex flex-col gap-y-6">
            <section className="flex flex-col gap-y-4">
              {groups.length === 0 ? (
                <p className="textinfolabel">
                  {t("settings.mcp.contents.methods.none")}
                </p>
              ) : (
                groups.map((group) => (
                  <div key={group.service} className="flex flex-col gap-y-2">
                    <h4 className="text-xs font-medium tracking-wide text-control-light">
                      {group.service}
                    </h4>
                    {/* The permission is rendered, not hovered. It used to sit
                        in a native title on a Badge, which is a span: a keyboard
                        user could never open it, and it was the only place the
                        permission appeared. */}
                    <div className="flex flex-col gap-y-1">
                      {group.methods.map((method) => (
                        <div
                          key={method.method}
                          className="grid grid-cols-[1fr_auto] items-baseline gap-x-4 text-xs"
                        >
                          <span
                            className={cn(
                              "font-mono",
                              method.class === MCPMethodClass.WRITE
                                ? "text-warning"
                                : "text-main"
                            )}
                          >
                            {method.operationId.split(".").pop()}
                          </span>
                          <span className="font-mono text-control-light">
                            {method.permission ||
                              t("settings.mcp.contents.methods.handler")}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                ))
              )}
            </section>

            {capability === MCPSetting_Capability.READ_ONLY && (
              <>
                <Separator />
                <section className="flex flex-col gap-y-3">
                  <div className="flex flex-col gap-y-1">
                    <h3 className="text-base font-medium">
                      {t("settings.mcp.contents.depth.title")}
                    </h3>
                    <p className="textinfolabel">
                      {t("settings.mcp.contents.depth.description")}
                    </p>
                  </div>
                  {depthGroups.map((group) => (
                    <div key={group.value} className="flex flex-col gap-y-1">
                      <p className="text-sm text-main">
                        {depthText(group.value)}
                      </p>
                      <p className="text-sm text-control-light">
                        {group.engines.map(engineLabel).join(", ")}
                      </p>
                    </div>
                  ))}
                  <p className="textinfolabel">
                    {t("settings.mcp.contents.depth.not-a-proof")}
                  </p>
                </section>
              </>
            )}

            <Separator />
            <section className="flex flex-col gap-y-3">
              <div className="flex flex-col gap-y-1">
                <h3 className="text-base font-medium">
                  {t("settings.mcp.contents.masking.title")}
                </h3>
                <p className="textinfolabel">
                  {ignoreMaskingExemptions
                    ? t("settings.mcp.contents.masking.description-on")
                    : t("settings.mcp.contents.masking.description-off")}
                </p>
              </div>
              {!info.dataMaskingAvailable && (
                <p className="text-sm text-warning">
                  {t("settings.mcp.policy.masking.unavailable")}
                </p>
              )}
              {maskingGroups.map((group) => (
                <div key={group.value} className="flex flex-col gap-y-1">
                  <p className="text-sm text-main">
                    {maskingText(group.value)}
                  </p>
                  <p className="text-sm text-control-light">
                    {group.engines.map(engineLabel).join(", ")}
                  </p>
                </div>
              ))}
            </section>

            {notes.length > 0 && (
              <>
                <Separator />
                <section className="flex flex-col gap-y-2">
                  <h3 className="text-base font-medium">
                    {t("settings.mcp.contents.notes.title")}
                  </h3>
                  {notes.map((engine) => (
                    <p key={engine.engine} className="text-sm text-control">
                      <span className="font-medium">
                        {engineLabel(engine.engine)}
                      </span>
                      {": "}
                      {noteText(engine.engine)}
                    </p>
                  ))}
                </section>
              </>
            )}
          </div>
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}
