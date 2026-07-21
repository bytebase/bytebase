import { useMemo } from "react";
import type { SQLResultSetV1 } from "@/types";
import type { QueryResult_PostgresError } from "@/types/proto-es/v1/sql_service_pb";

interface PostgresErrorProps {
  resultSet: SQLResultSetV1;
}

export function PostgresError({ resultSet }: PostgresErrorProps) {
  const errors = useMemo<QueryResult_PostgresError[]>(() => {
    const out: QueryResult_PostgresError[] = [];
    for (const r of resultSet.results) {
      if (r.detailedError?.case === "postgresError") {
        out.push(r.detailedError.value);
      }
    }
    return out;
  }, [resultSet.results]);

  return (
    <>
      {errors.map((error, i) => (
        <div
          key={i}
          className="text-sm grid gap-1 pl-8"
          style={{ gridTemplateColumns: "auto 1fr" }}
        >
          {error.detail && (
            <>
              <div>DETAIL:</div>
              <div>{error.detail}</div>
            </>
          )}
          {error.hint && (
            <>
              <div>HINT:</div>
              <div>{error.hint}</div>
            </>
          )}
          {error.where && (
            <>
              <div>WHERE:</div>
              <div>{error.where}</div>
            </>
          )}
        </div>
      ))}
    </>
  );
}
