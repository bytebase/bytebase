import { format, FormatOptions, supportedDialects } from "sql-formatter";

import { SQLDialect } from "../../types";

type FormatResult = {
  data: string;
  error: Error | null;
};

const formatSQL = (sql: string, dialect: SQLDialect): FormatResult => {
  if (!supportedDialects.includes(dialect)) {
    dialect = "mysql";
  }
  const options: Partial<FormatOptions> = {
    language: dialect as FormatOptions["language"],
  };

  try {
    const formatted = format(sql, options);
    return { data: formatted, error: null };
  } catch (error) {
    return { data: "", error: error as Error };
  }
};

export default formatSQL;
