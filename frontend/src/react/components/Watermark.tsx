import type { CSSProperties } from "react";
import { useMemo } from "react";
import {
  useCurrentUser,
  usePlanFeature,
  useServerInfo,
  useWorkspaceProfile,
} from "@/react/hooks/useAppState";
import { extractUserEmail } from "@/store/modules/v1/common";
import { UNKNOWN_USER_NAME } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

const USER_LAYER_CELL_W = 320;
const USER_LAYER_CELL_H = 200;
const USER_LAYER_FONT_SIZE = 16;
const USER_LAYER_LINE_STEP = 20;
const USER_LAYER_PADDING = 6;
const VERSION_LAYER_CELL = 128;
const VERSION_LAYER_FONT_SIZE = 16;

const baseLayerStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  pointerEvents: "none",
  backgroundRepeat: "repeat",
};

function makeWatermarkDataURL(opts: {
  content: string;
  fontSize: number;
  fontColor: string;
  rotateDeg: number;
  cellW: number;
  cellH: number;
}): string {
  const { content, fontSize, fontColor, rotateDeg, cellW, cellH } = opts;
  if (typeof document === "undefined" || !content) return "";
  const dpr = window.devicePixelRatio || 1;
  const canvas = document.createElement("canvas");
  canvas.width = cellW * dpr;
  canvas.height = cellH * dpr;
  const ctx = canvas.getContext("2d");
  if (!ctx) return "";
  ctx.scale(dpr, dpr);
  ctx.font = `${fontSize}px sans-serif`;
  ctx.fillStyle = fontColor;
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.translate(cellW / 2, cellH / 2);
  ctx.rotate((rotateDeg * Math.PI) / 180);
  ctx.fillText(content, 0, 0);
  return canvas.toDataURL();
}

export function Watermark() {
  const currentUser = useCurrentUser();
  const hasWatermarkFeature = usePlanFeature(PlanFeature.FEATURE_WATERMARK);
  const serverInfo = useServerInfo();
  const workspaceProfile = useWorkspaceProfile();

  const version = useMemo(() => {
    const v = serverInfo?.version ?? "";
    const commit = (serverInfo?.gitCommit ?? "").substring(0, 7);
    return `${v}-${commit}`;
  }, [serverInfo?.version, serverInfo?.gitCommit]);

  const userLines = useMemo(() => {
    if (
      !currentUser ||
      currentUser.name === UNKNOWN_USER_NAME ||
      !hasWatermarkFeature ||
      !workspaceProfile?.watermark
    ) {
      return [];
    }
    const uid = extractUserEmail(currentUser.name);
    const lines = [`${currentUser.title} (${uid})`];
    if (currentUser.email) lines.push(currentUser.email);
    return lines;
  }, [currentUser, hasWatermarkFeature, workspaceProfile?.watermark]);

  const versionDataURL = useMemo(
    () =>
      makeWatermarkDataURL({
        content: version,
        fontSize: VERSION_LAYER_FONT_SIZE,
        fontColor: "rgba(255, 128, 128, 0.01)",
        rotateDeg: 15,
        cellW: VERSION_LAYER_CELL,
        cellH: VERSION_LAYER_CELL,
      }),
    [version]
  );

  const userDataURLs = useMemo(
    () =>
      userLines.map((line) =>
        makeWatermarkDataURL({
          content: line,
          fontSize: USER_LAYER_FONT_SIZE,
          fontColor: "rgba(128, 128, 128, 0.1)",
          rotateDeg: -15,
          cellW: USER_LAYER_CELL_W,
          cellH: USER_LAYER_CELL_H,
        })
      ),
    [userLines]
  );

  return (
    <>
      {versionDataURL && (
        <div
          aria-hidden="true"
          style={{
            ...baseLayerStyle,
            zIndex: 10000000,
            backgroundImage: `url(${versionDataURL})`,
            backgroundSize: `${VERSION_LAYER_CELL}px ${VERSION_LAYER_CELL}px`,
            backgroundPosition: "24px 80px",
          }}
        />
      )}
      {userDataURLs.map((url, i) => {
        const base = USER_LAYER_CELL_H - USER_LAYER_PADDING;
        const yOffset = base - (userDataURLs.length - i) * USER_LAYER_LINE_STEP;
        return (
          <div
            key={i}
            aria-hidden="true"
            style={{
              ...baseLayerStyle,
              zIndex: 10000001,
              backgroundImage: `url(${url})`,
              backgroundSize: `${USER_LAYER_CELL_W}px ${USER_LAYER_CELL_H}px`,
              backgroundPosition: `0px ${yOffset}px`,
            }}
          />
        );
      })}
    </>
  );
}
