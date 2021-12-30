import { format, FormatOptions } from "sql-formatter";

import { SqlLanguage } from "../../types";

type FormatResult = {
  data: string;
  error: Error | null;
};

const formatSQL = (sql: string, lang: SqlLanguage): FormatResult => {
  const options: FormatOptions = {
    language: lang,
  };
  try {
    const formatted = format(sql, options);
    return { data: formatted, error: null };
  } catch (error) {
    return { data: "", error: error as Error };
  }
};

export default formatSQL;
