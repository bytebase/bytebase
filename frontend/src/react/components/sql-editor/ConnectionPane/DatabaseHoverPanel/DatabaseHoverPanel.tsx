import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import type { Position, SQLEditorTreeNode } from "@/types";
import {
  getDatabaseProject,
  getInstanceResource,
  instanceV1Name,
  minmax,
  projectV1Name,
} from "@/utils";
import { useHoverState } from "./hover-state";

type Props = {
  readonly offsetX: number;
  readonly offsetY: number;
  readonly margin: number;
  readonly onClickOutside?: (e: MouseEvent) => void;
};

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/DatabaseHoverPanel/DatabaseHoverPanel.vue.
 * Floating panel anchored at the hovered tree row showing database context
 * (environment, instance, project, labels). Clamps the y coordinate so the
 * panel stays fully within the viewport.
 */
export function DatabaseHoverPanel({
  offsetX,
  offsetY,
  margin,
  onClickOutside,
}: Props) {
  const { t } = useTranslation();
  const editorStore = useSQLEditorVueState();
  const { state, position, update } = useHoverState();

  const hasProjectContext = useVueState(() => !!editorStore.project);

  const popoverRef = useRef<HTMLDivElement | null>(null);
  const [popoverHeight, setPopoverHeight] = useState(0);

  useLayoutEffect(() => {
    const el = popoverRef.current;
    if (!el) return;
    setPopoverHeight(el.getBoundingClientRect().height);
    const ro = new ResizeObserver(() => {
      setPopoverHeight(el.getBoundingClientRect().height);
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, [state]);

  const database = useMemo(() => {
    if (!state?.node) return undefined;
    const { type, target } = state.node.meta;
    if (type !== "database") return undefined;
    return target as SQLEditorTreeNode<"database">["meta"]["target"];
  }, [state]);

  const show = state !== undefined && position.x !== 0 && position.y !== 0;

  const displayPosition = useMemo<Position>(() => {
    const p: Position = {
      x: position.x + offsetX,
      y: position.y + offsetY,
    };
    if (typeof window !== "undefined") {
      const yMin = margin;
      const yMax = window.innerHeight - popoverHeight - margin;
      p.y = minmax(p.y, yMin, yMax);
    }
    return p;
  }, [position, offsetX, offsetY, margin, popoverHeight]);

  useEffect(() => {
    if (!show || !onClickOutside) return;
    const handler = (e: MouseEvent) => {
      const el = popoverRef.current;
      if (el && !el.contains(e.target as Node)) {
        onClickOutside(e);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [show, onClickOutside]);

  if (!database && !show) return null;

  const panel = (
    <div
      ref={popoverRef}
      className={cn(
        "fixed border border-gray-100 rounded-sm bg-white p-2 shadow-sm transition-all text-sm",
        LAYER_SURFACE_CLASS,
        !show && "pointer-events-none opacity-0"
      )}
      style={{
        left: `${displayPosition.x}px`,
        top: `${displayPosition.y}px`,
      }}
      onMouseEnter={() => update(state, "before", 50)}
      onMouseLeave={() => update(undefined, "after")}
    >
      {database ? (
        <div
          className="grid min-w-56 max-w-[18rem] gap-x-2 gap-y-1"
          style={{ gridTemplateColumns: "auto 1fr" }}
        >
          <div className="contents">
            <div className="text-gray-500 font-medium">
              {t("common.environment")}
            </div>
            <div className="text-main text-right flex justify-end">
              <EnvironmentLabel
                environmentName={database.effectiveEnvironment}
              />
            </div>
          </div>
          <div className="contents">
            <div className="text-gray-500 font-medium">
              {t("common.instance")}
            </div>
            <div className="text-main text-right truncate">
              {instanceV1Name(getInstanceResource(database))}
            </div>
          </div>
          {!hasProjectContext && (
            <div className="contents">
              <div className="text-gray-500 font-medium">
                {t("common.project")}
              </div>
              <div className="text-main text-right truncate">
                {projectV1Name(getDatabaseProject(database))}
              </div>
            </div>
          )}
          <div className="contents">
            <div className="text-gray-500 font-medium">
              {t("common.labels")}
            </div>
            <div className="text-main flex flex-row justify-end flex-wrap gap-1">
              {Object.entries(database.labels).map(([key, value]) => (
                <div
                  key={key}
                  className="text-xs py-px px-1 bg-gray-200/75 rounded-xs"
                >
                  <span>{key}</span>
                  {value ? (
                    <>
                      <span>:</span>
                      <span>{value}</span>
                    </>
                  ) : null}
                </div>
              ))}
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );

  if (typeof document === "undefined") return null;
  return createPortal(panel, getLayerRoot("overlay"));
}
