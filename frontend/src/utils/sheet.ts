import Long from "long";

export const getStatementSize = (statement: string): Long => {
  return Long.fromNumber(new TextEncoder().encode(statement).length);
};
