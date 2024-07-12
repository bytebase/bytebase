import { Range } from "monaco-editor";
import { isRef, unref, watch } from "vue";
import { h } from "vue";
import { t } from "@/plugins/i18n";
import { pushNotification } from "@/store";
import type { Language, MaybeRef, SQLDialect } from "@/types";
import { minmax } from "@/utils";
import LearnMoreLink from "../LearnMoreLink.vue";
import sqlFormatter from "./sqlFormatter";
import type { IStandaloneCodeEditor } from "./types";

// Max retires in a retry serial. Will be reset after a success connection
export const MAX_RETRIES = 5;
// Progressive delay in a retry serial. Avoiding to flood the server.
export const RECONNECTION_DELAY = {
  max: 1000,
  min: 100,
  growth: 1.5,
};
// Timeout to setup connection in EACH attempt
export const WEBSOCKET_TIMEOUT = 5000;

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
  }
  // A simple fallback
  console.warn("unexpected language", lang);
  return "sql";
};

export const useEditorContextKey = <
  T extends string | number | boolean | null | undefined,
>(
  editor: IStandaloneCodeEditor,
  key: string,
  valueOrRef: MaybeRef<T>
) => {
  const contextKey = editor.createContextKey<T>(key, unref(valueOrRef));
  if (isRef(valueOrRef)) {
    watch(valueOrRef, (value) => contextKey?.set(value));
  }
  return contextKey;
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
  const { data, error } = await sqlFormatter(sql, dialect);
  if (error) {
    return;
  }
  const pos = editor.getPosition();

  trySetContentWithUndo(editor, data, "Format content");

  if (pos) {
    // Not that smart but best efforts to keep the cursor position
    editor.setPosition(pos);
  }
};

export const createUrl = (
  host: string,
  path: string,
  searchParams: Record<string, any> = {},
  secure: boolean = location.protocol === "https:"
) => {
  const protocol = secure ? "wss" : "ws";
  const url = new URL(`${protocol}://${host}${path}`);

  for (const [key, value] of Object.entries(searchParams)) {
    const v = value instanceof Array ? value.join(",") : value;
    if (v) {
      url.searchParams.set(key, v);
    }
  }
  return url;
};

const extractErrorMessage = (err: any) => {
  if (typeof err === "string") {
    return err;
  }
  if (typeof err.message === "string") {
    return err.message;
  }
  return String(err);
};

export const errorNotification = (err: unknown) => {
  pushNotification({
    module: "bytebase",
    style: "CRITICAL",
    title: messages.title(),
    description: () => {
      const message = extractErrorMessage(err);
      return [
        h("p", {}, messages.description()),
        message ? h("p", {}, message) : null,
        h(LearnMoreLink, {
          url: "https://www.bytebase.com/docs/administration/production-setup/#enable-https-and-websocket",
        }),
      ];
    },
  });
};

export const progressiveDelay = (serial: number) => {
  if (serial === 0) {
    return 0;
  }
  return minmax(
    RECONNECTION_DELAY.min * Math.pow(RECONNECTION_DELAY.growth, serial - 1),
    RECONNECTION_DELAY.min,
    RECONNECTION_DELAY.max
  );
};
