import { Range } from "monaco-editor";
import { t } from "@/plugins/i18n";
import { pushNotification } from "@/store";
import type { Language, SQLDialect } from "@/types";
import { callCssVariable, escapeMarkdown, minmax } from "@/utils";
import { formatSQL } from "./sqlFormatter";
import type { IStandaloneCodeEditor, Selection } from "./types";

export const MAX_RETRIES = 5;
export const RECONNECTION_DELAY = {
  max: 1000,
  min: 100,
  growth: 1.5,
};
export const WEBSOCKET_TIMEOUT = 5000;
export const WEBSOCKET_HEARTBEAT_INTERVAL = 10 * 1000;

export const messages = {
  title: () => t("sql-editor.web-socket.errors.title"),
  description: () => t("sql-editor.web-socket.errors.description"),
  disconnected: () => t("sql-editor.web-socket.errors.disconnected"),
};

export const extensionNameOfLanguage = (lang: Language) => {
  switch (lang) {
    case "sql":
      return "sql";
    case "javascript":
      return "js";
    case "redis":
      return "redis";
    case "json":
      return "json";
  }
  return "sql";
};

export const trySetContentWithUndo = (
  editor: IStandaloneCodeEditor,
  content: string,
  source: string | undefined = undefined
) => {
  editor.executeEdits(source, [
    {
      range: new Range(1, 1, Number.MAX_SAFE_INTEGER, 1),
      text: "",
      forceMoveMarkers: true,
    },
    {
      range: new Range(1, 1, 1, 1),
      text: content,
      forceMoveMarkers: true,
    },
  ]);
};

export const formatEditorContent = async (
  editor: IStandaloneCodeEditor,
  dialect: SQLDialect | undefined
) => {
  const model = editor.getModel();
  if (!model) return;
  const sql = model.getValue();
  const { data, error } = await formatSQL(sql, dialect);
  if (error) return;
  const pos = editor.getPosition();
  trySetContentWithUndo(editor, data, "Format content");
  if (pos) {
    editor.setPosition(pos);
  }
};

export const createUrl = (
  host: string,
  path: string,
  secure: boolean = location.protocol === "https:"
) => {
  const protocol = secure ? "wss" : "ws";
  return new URL(`${protocol}://${host}${path}`);
};

const extractErrorMessage = (err: unknown) => {
  if (typeof err === "string") return err;
  if (typeof (err as Error).message === "string") {
    return (err as Error).message;
  }
  return String(err);
};

export const errorNotification = (err: unknown) => {
  const message = extractErrorMessage(err);
  pushNotification({
    module: "bytebase",
    style: "CRITICAL",
    title: messages.title(),
    description: `${messages.description()}${message ? ` ${message}` : ""}`,
  });
};

export const progressiveDelay = (serial: number) => {
  if (serial === 0) return 0;
  return minmax(
    RECONNECTION_DELAY.min * Math.pow(RECONNECTION_DELAY.growth, serial - 1),
    RECONNECTION_DELAY.min,
    RECONNECTION_DELAY.max
  );
};

export const positionWithOffset = (
  line: number,
  column: number,
  selection?: Selection | null
) => {
  if (!selection || selection.isEmpty()) {
    return [line, column];
  }
  const pos = selection.getStartPosition().delta(line - 1, column - 1);
  return [pos.lineNumber, pos.column];
};

export const buildAdviceHoverMessage = (advice: {
  severity: "ERROR" | "WARNING";
  message: string;
  source?: string;
}) => {
  const colors = {
    WARNING: callCssVariable("--color-warning"),
    ERROR: callCssVariable("--color-error"),
  };
  const parts: string[] = [];
  parts.push(
    `<span style="color:${colors[advice.severity]};">[${advice.severity}]</span>`
  );
  if (advice.source) {
    parts.push(` ${escapeMarkdown(advice.source)}`);
  }
  parts.push(`\n${escapeMarkdown(advice.message)}`);
  return parts.join("");
};
