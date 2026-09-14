import { Engine } from "@/types/proto-es/v1/common_pb";

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

export const createExplainToken = ({
  statement,
  explain,
  engine,
}: StoredExplain): string => {
  const token = `explain-${tokenSuffix()}`;

  const json = JSON.stringify({ statement, explain, engine });
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
