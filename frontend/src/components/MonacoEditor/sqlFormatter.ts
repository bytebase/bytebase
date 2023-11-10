import { format, FormatOptionsWithLanguage } from "sql-formatter";
import { SQLDialect } from "../../types";

type FormatResult = {
  data: string;
  error: Error | null;
};

type FormatterLanguage = FormatOptionsWithLanguage["language"];

const convertDialectToFormatterLanguage = (
  dialect: SQLDialect | undefined
): FormatterLanguage => {
  if (dialect === "MYSQL" || dialect === "TIDB" || dialect === "OCEANBASE")
    return "mysql";
  if (dialect === "POSTGRES") return "postgresql";
  if (dialect === "SNOWFLAKE") return "snowflake";
  return "sql";
};

const formatSQL = (
  sql: string,
  dialect: SQLDialect | undefined
): FormatResult => {
  const options: Partial<FormatOptionsWithLanguage> = {
    language: convertDialectToFormatterLanguage(dialect),
  };

  try {
    const formatted = format(sql, options);
    return { data: formatted, error: null };
  } catch (error) {
    return { data: "", error: error as Error };
  }
};

export default formatSQL;
