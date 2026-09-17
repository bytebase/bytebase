import { Engine } from "@/types/proto-es/v1/common_pb";

/**
 * The engines the query-plan visualizer draws, each with the explain output it
 * reads, named as `QueryOption.ExplainFormat` names it. The SQL editor offers
 * Visualize for exactly these engines and hands over the plan in this format;
 * the visualizer keys its parsers on the same table, so the two cannot drift.
 */
export const VISUALIZER_EXPLAIN_FORMATS = {
  [Engine.POSTGRES]: "JSON",
  [Engine.MSSQL]: "XML",
  [Engine.SPANNER]: "JSON",
} as const;

export type VisualizerEngine = keyof typeof VISUALIZER_EXPLAIN_FORMATS;

export const isVisualizerEngine = (
  engine: Engine
): engine is VisualizerEngine => engine in VISUALIZER_EXPLAIN_FORMATS;

export type StoredExplain = {
  statement: string;
  explain: string;
  engine: Engine;
};

/**
 * Deliberately not `randomString` from `./util`: the visualizer is a separate
 * entry that imports this module and nothing else of the app, and `./util`
 * drags dayjs, semver, DOMPurify and the i18n bundle in behind it.
 *
 * The token only has to be unique within one tab's sessionStorage — it is a
 * lookup key, not a secret.
 */
const tokenSuffix = (): string => {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join(
    ""
  );
};

const TOKEN_PREFIX = "explain-";

/**
 * Drops the plans this tab stashed earlier.
 *
 * A plan is read once, by the tab this token opens, and `window.open` gives
 * that tab its own copy of sessionStorage — so the opener's copy is dead weight
 * the moment the visualizer has it. Left to accumulate, a few hundred-kilobyte
 * plans fill the origin's quota and every later click fails.
 */
const dropPreviousExplains = () => {
  const stale: string[] = [];
  for (let i = 0; i < sessionStorage.length; i++) {
    const key = sessionStorage.key(i);
    if (key?.startsWith(TOKEN_PREFIX)) stale.push(key);
  }
  for (const key of stale) sessionStorage.removeItem(key);
};

export const createExplainToken = ({
  statement,
  explain,
  engine,
}: StoredExplain): string => {
  const token = `${TOKEN_PREFIX}${tokenSuffix()}`;

  const json = JSON.stringify({ statement, explain, engine });
  dropPreviousExplains();
  sessionStorage.setItem(token, json);

  return token;
};

export const readExplainFromToken = (token: string) => {
  if (!token) return undefined;
  try {
    const json = sessionStorage.getItem(token) || "{}";
    const obj = JSON.parse(json) as StoredExplain;
    if (!obj) return undefined;
    if (typeof obj.statement === "string" && typeof obj.explain === "string") {
      return obj;
    }
    return undefined;
  } catch {
    return undefined;
  }
};
