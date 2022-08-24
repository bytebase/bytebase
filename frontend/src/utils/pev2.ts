import { randomString } from "./util";

export const createExplainToken = (
  statement: string,
  explain: string
): string => {
  const token = `pev2-${randomString(32)}`;

  const json = JSON.stringify({ statement, explain });
  sessionStorage.setItem(token, json);

  return token;
};

export type StoredExplain = { statement: string; explain: string };

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
