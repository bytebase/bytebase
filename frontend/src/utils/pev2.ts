import { Engine } from "@/types/proto-es/v1/common_pb";
import { randomString } from "./util";

export type StoredExplain = {
  statement: string;
  explain: string;
  engine: Engine;
};

export const createExplainToken = ({
  statement,
  explain,
  engine,
}: StoredExplain): string => {
  const token = `pev2-${randomString(32)}`;

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
