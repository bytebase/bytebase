import { format, FormatOptions } from "sql-formatter";

import { SQLDialect } from "../../types";

type FormatResult = {
  data: string;
  error: Error | null;
};

const formatSQL = (sql: string, dialect: SQLDialect): FormatResult => {
  const options: FormatOptions = {
    language: dialect,
  };
  if (dialect !== "mysql" && dialect !== "postgresql") {
    options.language = "mysql";
  }

  try {
    const formatted = format(sql, options);
    return { data: formatted, error: null };
  } catch (error) {
    return { data: "", error: error as Error };
  }
};

export default formatSQL;
