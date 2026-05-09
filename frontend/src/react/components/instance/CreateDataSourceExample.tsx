import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";
import {
  DATASOURCE_ADMIN_USER_NAME,
  DATASOURCE_READONLY_USER_NAME,
  languageOfEngineV1,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType,
  DataSourceType,
} from "@/types/proto-es/v1/instance_service_pb";

export interface CreateDataSourceExampleProps {
  engine: Engine;
  dataSourceType: DataSourceType;
  authenticationType: DataSource_AuthenticationType;
  createInstanceFlag: boolean;
  className?: string;
}

function getGrantStatement(
  engine: Engine,
  dataSourceType: DataSourceType,
  authenticationType: DataSource_AuthenticationType
): string {
  if (dataSourceType === DataSourceType.ADMIN) {
    const createUserStatement = `CREATE USER ${DATASOURCE_ADMIN_USER_NAME}@'%' IDENTIFIED BY 'YOUR_DB_PWD';`;
    switch (engine) {
      case Engine.MYSQL:
        if (
          authenticationType ===
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        ) {
          return `GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nRELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE \n/*!80000 , SET_USER_ID */\nON *.* to ${DATASOURCE_ADMIN_USER_NAME}@'%';`;
        } else if (
          authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
        ) {
          return `CREATE USER ${DATASOURCE_ADMIN_USER_NAME}@'%' IDENTIFIED WITH AWSAuthenticationPlugin AS 'RDS';\n\nALTER USER '${DATASOURCE_ADMIN_USER_NAME}'@'%' REQUIRE SSL;\n\nGRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nRELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE \n/*!80000 , SET_USER_ID */\nON *.* to ${DATASOURCE_ADMIN_USER_NAME}@'%';`;
        }
        return `${createUserStatement}\n\nGRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nRELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE \n/*!80000 , SET_USER_ID */\nON *.* to ${DATASOURCE_ADMIN_USER_NAME}@'%';`;
      case Engine.TIDB:
        return `${createUserStatement}\n\nGRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nLOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE \nON *.* to ${DATASOURCE_ADMIN_USER_NAME}@'%';`;
      case Engine.MARIADB:
        return `${createUserStatement}\n\nGRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nRELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE \nON *.* to ${DATASOURCE_ADMIN_USER_NAME}@'%';`;
      case Engine.OCEANBASE:
        return `${createUserStatement}\n\nGRANT ALTER, CREATE, CREATE VIEW, DELETE, DROP, INDEX, INSERT, \nPROCESS, SELECT, SHOW DATABASES, SHOW VIEW, UPDATE, USAGE, \nREPLICATION CLIENT, REPLICATION SLAVE \nON *.* to ${DATASOURCE_ADMIN_USER_NAME}@'%';`;
      case Engine.CLICKHOUSE:
        return `CREATE USER ${DATASOURCE_ADMIN_USER_NAME} IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT ALL ON *.* TO ${DATASOURCE_ADMIN_USER_NAME} WITH GRANT OPTION;`;
      case Engine.SNOWFLAKE:
        return [
          "-- Option 1: grant ACCOUNTADMIN role",
          "",
          `CREATE OR REPLACE USER ${DATASOURCE_ADMIN_USER_NAME} PASSWORD = 'YOUR_DB_PWD'`,
          'DEFAULT_ROLE = "ACCOUNTADMIN"',
          "DEFAULT_WAREHOUSE = 'YOUR_COMPUTE_WAREHOUSE';",
          "",
          `GRANT ROLE "ACCOUNTADMIN" TO USER ${DATASOURCE_ADMIN_USER_NAME};`,
          "",
          "-- Option 2: grant more granular privileges",
          "",
          "CREATE OR REPLACE ROLE BYTEBASE;",
          "",
          "-- If using non-enterprise edition, the following commands may encounter error likes 'Unsupported feature GRANT/REVOKE APPLY TAG', you can skip those unsupported GRANTs.",
          "-- Grant the least privileges required by Bytebase ",
          "",
          "GRANT CREATE DATABASE, EXECUTE TASK, IMPORT SHARE, APPLY MASKING POLICY, APPLY ROW ACCESS POLICY, APPLY TAG ON ACCOUNT TO ROLE BYTEBASE;",
          "",
          "GRANT IMPORTED PRIVILEGES ON DATABASE SNOWFLAKE TO ROLE BYTEBASE;",
          "",
          'GRANT USAGE ON WAREHOUSE "YOUR_COMPUTE_WAREHOUSE" TO ROLE BYTEBASE;',
          "",
          "CREATE OR REPLACE USER BYTEBASE",
          "  PASSWORD = 'YOUR_PWD'",
          '  DEFAULT_ROLE = "BYTEBASE"',
          '  DEFAULT_WAREHOUSE = "YOUR_COMPUTE_WAREHOUSE";',
          "",
          'GRANT ROLE "BYTEBASE" TO USER BYTEBASE;',
          "",
          'GRANT ROLE "BYTEBASE" TO ROLE SYSADMIN;',
          "",
          "-- For each database to be managed by Bytebase, you need to grant the following privileges",
          "",
          "GRANT ALL PRIVILEGES ON DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL SCHEMAS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE SCHEMAS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "-- Then, grant the required schema object privileges on the database to the Bytebase role",
          "GRANT ALL PRIVILEGES ON ALL EVENT TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE EVENT TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL EXTERNAL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE EXTERNAL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL FILE FORMATS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE FILE FORMATS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE FUNCTIONS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL MATERIALIZED VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE MATERIALIZED VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL PROCEDURES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE PROCEDURES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE SEQUENCES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL STAGES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE STAGES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL STREAMS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE STREAMS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL TASKS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE TASKS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "-- ALERT and POLICY are the features in Snowflake Enterprise Edition, you can skip those GRANTs if you are using standard edition.",
          "GRANT ALL PRIVILEGES ON ALL ALERTS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE ALERTS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL PASSWORD POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE PASSWORD POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL ROW ACCESS POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE ROW ACCESS POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL MASKING POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE MASKING POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "GRANT ALL PRIVILEGES ON ALL SESSION POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "GRANT ALL PRIVILEGES ON FUTURE SESSION POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
          "",
          "-- PIPE are not allowed to be bulk granted, you need to grant them one by one.",
          "GRANT ALL PRIVILEGES ON PIPE {{PIPE_NAME}} IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;",
        ].join("\n");
      case Engine.POSTGRES:
        if (
          authenticationType ===
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        ) {
          return `GRANT pg_write_all_data TO "${DATASOURCE_ADMIN_USER_NAME}@{project-id}.iam";\n\n-- If you need to create databases via Bytebase\nALTER USER "${DATASOURCE_ADMIN_USER_NAME}@{project-id}.iam" WITH CREATEDB;`;
        } else if (
          authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
        ) {
          return `CREATE USER ${DATASOURCE_ADMIN_USER_NAME};\n\nGRANT rds_iam TO ${DATASOURCE_ADMIN_USER_NAME};\n\nGRANT pg_write_all_data TO ${DATASOURCE_ADMIN_USER_NAME};\n\n-- If you need to create databases via Bytebase\nALTER USER ${DATASOURCE_ADMIN_USER_NAME} WITH CREATEDB;`;
        }
        return `CREATE USER ${DATASOURCE_ADMIN_USER_NAME} WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';\n\nALTER USER ${DATASOURCE_ADMIN_USER_NAME} WITH SUPERUSER;`;
      case Engine.REDSHIFT:
        return `CREATE USER ${DATASOURCE_ADMIN_USER_NAME} WITH PASSWORD 'YOUR_DB_PWD' CREATEUSER CREATEDB;`;
      case Engine.MONGODB:
        return `use admin;\ndb.createUser({\n  user: "${DATASOURCE_ADMIN_USER_NAME}",\n  pwd: "YOUR_DB_PWD",\n  roles: [\n    {role: "readWriteAnyDatabase", db: "admin"},\n    {role: "dbAdminAnyDatabase", db: "admin"},\n    {role: "userAdminAnyDatabase", db: "admin"}\n  ]\n});`;
      case Engine.SPANNER:
        return "";
      case Engine.BIGQUERY:
        return "";
      case Engine.DYNAMODB:
        return "";
      case Engine.COCKROACHDB:
        return "";
      case Engine.REDIS:
        return `ACL SETUSER ${DATASOURCE_ADMIN_USER_NAME} on >YOUR_DB_PWD +@all &*`;
      case Engine.MSSQL:
        return `-- If you use Cloud RDS, you need to checkout their documentation for setting up a semi-super privileged user.\nCREATE LOGIN ${DATASOURCE_ADMIN_USER_NAME} WITH PASSWORD = 'YOUR_DB_PWD';\nALTER SERVER ROLE sysadmin ADD MEMBER ${DATASOURCE_ADMIN_USER_NAME};`;
      case Engine.ORACLE:
        return `-- If you use Cloud RDS, you need to checkout their documentation for setting up a semi-super privileged user.\nCREATE USER ${DATASOURCE_ADMIN_USER_NAME} IDENTIFIED BY 'YOUR_DB_PWD';\nGRANT ALL PRIVILEGES TO ${DATASOURCE_ADMIN_USER_NAME};`;
    }
  } else {
    const mysqlReadonlyStatement = `CREATE USER ${DATASOURCE_READONLY_USER_NAME}@'%' IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT SELECT, SHOW DATABASES, SHOW VIEW, USAGE ON *.* to ${DATASOURCE_READONLY_USER_NAME}@'%';`;
    switch (engine) {
      case Engine.MYSQL:
        if (
          authenticationType ===
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        ) {
          return `GRANT SELECT, SHOW DATABASES, SHOW VIEW, USAGE ON *.* to ${DATASOURCE_READONLY_USER_NAME}@'%';`;
        } else if (
          authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
        ) {
          return `CREATE USER ${DATASOURCE_READONLY_USER_NAME}@'%' IDENTIFIED WITH AWSAuthenticationPlugin AS 'RDS';\n\nALTER USER '${DATASOURCE_READONLY_USER_NAME}'@'%' REQUIRE SSL;\n\nGRANT SELECT, SHOW DATABASES, SHOW VIEW, USAGE ON *.* to ${DATASOURCE_READONLY_USER_NAME}@'%';`;
        }
        return mysqlReadonlyStatement;
      case Engine.TIDB:
      case Engine.OCEANBASE:
        return mysqlReadonlyStatement;
      case Engine.CLICKHOUSE:
        return `CREATE USER ${DATASOURCE_READONLY_USER_NAME} IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT SHOW TABLES, SELECT ON database.* TO ${DATASOURCE_READONLY_USER_NAME};`;
      case Engine.SNOWFLAKE:
        return [
          "-- Option 1: grant ACCOUNTADMIN role",
          "",
          `CREATE OR REPLACE USER ${DATASOURCE_READONLY_USER_NAME} PASSWORD = 'YOUR_DB_PWD'`,
          'DEFAULT_ROLE = "ACCOUNTADMIN"',
          "DEFAULT_WAREHOUSE = 'YOUR_COMPUTE_WAREHOUSE';",
          "",
          `GRANT ROLE "ACCOUNTADMIN" TO USER ${DATASOURCE_READONLY_USER_NAME};`,
          "",
          "-- Option 2: grant more granular privileges",
          "",
          "CREATE OR REPLACE ROLE BYTEBASE_READER;",
          "",
          "-- If using non-enterprise edition, the following commands may encounter error likes 'Unsupported feature GRANT/REVOKE APPLY TAG', you can skip those unsupported GRANTs.",
          "-- Grant the least privileges required by Bytebase ",
          "",
          "GRANT IMPORT SHARE TO ROLE BYTEBASE_READER;",
          "",
          "GRANT IMPORTED PRIVILEGES ON DATABASE SNOWFLAKE TO ROLE BYTEBASE_READER;",
          "",
          'GRANT USAGE ON WAREHOUSE "YOUR_COMPUTE_WAREHOUSE" TO ROLE BYTEBASE_READER;',
          "",
          "CREATE OR REPLACE USER BYTEBASE_READER",
          "  PASSWORD = 'YOUR_PWD'",
          '  DEFAULT_ROLE = "BYTEBASE_READER"',
          '  DEFAULT_WAREHOUSE = "YOUR_COMPUTE_WAREHOUSE";',
          "",
          'GRANT ROLE "BYTEBASE_READER" TO USER BYTEBASE_READER;',
          "",
          'GRANT ROLE "BYTEBASE_READER" TO ROLE SYSADMIN;',
          "",
          "-- For each database to be managed by Bytebase, you need to grant the following privileges",
          "",
          "GRANT ALL PRIVILEGES ON DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL SCHEMAS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE SCHEMAS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "-- Then, grant the required schema object privileges on the database to the Bytebase role",
          "GRANT ALL PRIVILEGES ON ALL EVENT TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE EVENT TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL EXTERNAL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE EXTERNAL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL FILE FORMATS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE FILE FORMATS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE FUNCTIONS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL MATERIALIZED VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE MATERIALIZED VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL PROCEDURES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE PROCEDURES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE SEQUENCES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL STAGES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE STAGES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL STREAMS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE STREAMS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL TASKS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE TASKS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "-- ALERT and POLICY are the features in Snowflake Enterprise Edition, you can skip those GRANTs if you are using standard edition.",
          "GRANT ALL PRIVILEGES ON ALL ALERTS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE ALERTS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL PASSWORD POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE PASSWORD POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL ROW ACCESS POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE ROW ACCESS POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL MASKING POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE MASKING POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "GRANT ALL PRIVILEGES ON ALL SESSION POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "GRANT ALL PRIVILEGES ON FUTURE SESSION POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
          "",
          "-- PIPE are not allowed to be bulk granted, you need to grant them one by one.",
          "GRANT ALL PRIVILEGES ON PIPE {{PIPE_NAME}} IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;",
        ].join("\n");
      case Engine.POSTGRES:
        if (
          authenticationType ===
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        ) {
          return `GRANT pg_read_all_data TO "${DATASOURCE_READONLY_USER_NAME}@{project-id}.iam";`;
        } else if (
          authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
        ) {
          return `CREATE USER ${DATASOURCE_READONLY_USER_NAME};\n\nGRANT rds_iam TO ${DATASOURCE_READONLY_USER_NAME};\n\nGRANT pg_read_all_data TO ${DATASOURCE_READONLY_USER_NAME};`;
        }
        return `CREATE USER ${DATASOURCE_READONLY_USER_NAME} WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';\n\nGRANT pg_read_all_data TO ${DATASOURCE_READONLY_USER_NAME};`;
      case Engine.MONGODB:
        return `use admin;\ndb.createUser({\n  user: "${DATASOURCE_READONLY_USER_NAME}",\n  pwd: passwordPrompt(),\n  roles: [\n    {role: "readAnyDatabase", db: "admin"}\n  ]\n});`;
      case Engine.SPANNER:
        return "";
      case Engine.BIGQUERY:
        return "";
      case Engine.DYNAMODB:
        return "";
      case Engine.COCKROACHDB:
        return "";
      case Engine.REDIS:
        return `ACL SETUSER ${DATASOURCE_READONLY_USER_NAME} on >YOUR_DB_PWD +@read &*`;
    }
  }
  return "";
}

function EngineSpecificDescription({
  engine,
  authenticationType,
  userName,
}: {
  engine: Engine;
  authenticationType: DataSource_AuthenticationType;
  userName: string;
}) {
  const { t } = useTranslation();

  if (
    engine === Engine.MYSQL ||
    engine === Engine.TIDB ||
    engine === Engine.OCEANBASE
  ) {
    if (
      authenticationType === DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
    ) {
      return (
        <span>
          {t("instance.sentence.google-cloud-sql.mysql.template", {
            user: userName,
          })}
        </span>
      );
    }
    if (authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM) {
      return (
        <span>
          {t("instance.sentence.aws-rds.mysql.template", {
            user: userName,
          })}
        </span>
      );
    }
    return (
      <span>
        {t("instance.sentence.create-user-example.mysql.template", {
          user: userName,
          password: "YOUR_DB_PWD",
        })}
      </span>
    );
  }

  if (engine === Engine.CLICKHOUSE) {
    return (
      <p>
        {t("instance.sentence.create-user-example.clickhouse.template", {
          user: userName,
          password: "YOUR_DB_PWD",
          link: t(
            "instance.sentence.create-user-example.clickhouse.sql-driven-workflow"
          ),
        })}
      </p>
    );
  }

  if (engine === Engine.POSTGRES) {
    return (
      <>
        <Alert
          variant="warning"
          className="my-2"
          description={t(
            "instance.sentence.create-user-example.postgresql.warn"
          )}
        />
        {authenticationType ===
        DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM ? (
          <p>
            {t("instance.sentence.google-cloud-sql.postgresql.template", {
              user: userName,
            })}
          </p>
        ) : authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM ? (
          <p>
            {t("instance.sentence.aws-rds.postgresql.template", {
              user: userName,
            })}
          </p>
        ) : (
          <p>
            {t("instance.sentence.create-user-example.postgresql.template", {
              user: userName,
              password: "YOUR_DB_PWD",
            })}
          </p>
        )}
      </>
    );
  }

  if (engine === Engine.SNOWFLAKE) {
    return (
      <p>
        {t("instance.sentence.create-user-example.snowflake.template", {
          user: userName,
          password: "YOUR_DB_PWD",
          warehouse: "YOUR_COMPUTE_WAREHOUSE",
        })}
      </p>
    );
  }

  if (engine === Engine.REDIS) {
    return (
      <p>
        {t("instance.sentence.create-user-example.redis.template", {
          user: userName,
          password: "YOUR_DB_PWD",
        })}
      </p>
    );
  }

  return null;
}

export function CreateDataSourceExample({
  engine,
  dataSourceType,
  authenticationType,
  createInstanceFlag,
  className,
}: CreateDataSourceExampleProps) {
  const { t } = useTranslation();
  const [showExample, setShowExample] = useState(createInstanceFlag);

  const isEngineUsingSQL = useMemo(
    () => languageOfEngineV1(engine) === "sql",
    [engine]
  );

  const userName = useMemo(
    () =>
      dataSourceType === DataSourceType.ADMIN
        ? DATASOURCE_ADMIN_USER_NAME
        : DATASOURCE_READONLY_USER_NAME,
    [dataSourceType]
  );

  const grantStatement = useMemo(
    () => getGrantStatement(engine, dataSourceType, authenticationType),
    [engine, dataSourceType, authenticationType]
  );

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(grantStatement);
  }, [grantStatement]);

  const toggleExample = useCallback(() => {
    setShowExample((prev) => !prev);
  }, []);

  if (!grantStatement) {
    return null;
  }

  const descriptionText = isEngineUsingSQL
    ? dataSourceType === DataSourceType.ADMIN
      ? t("instance.sentence.create-admin-user")
      : t("instance.sentence.create-readonly-user")
    : dataSourceType === DataSourceType.ADMIN
      ? t("instance.sentence.create-admin-user-non-sql")
      : t("instance.sentence.create-readonly-user-non-sql");

  return (
    <div className={`w-full flex flex-col justify-start ${className ?? ""}`}>
      <p className="w-full text-sm text-gray-500">
        {descriptionText}
        {!createInstanceFlag && (
          <span
            className="normal-link select-none ml-1 cursor-pointer"
            onClick={toggleExample}
          >
            {t("instance.show-how-to-create")}
          </span>
        )}
      </p>
      {showExample && (
        <div className="text-sm text-main">
          <EngineSpecificDescription
            engine={engine}
            authenticationType={authenticationType}
            userName={userName}
          />
          <div className="mt-2 flex flex-row">
            <pre className="flex-1 min-w-0 w-full inline-flex items-center px-3 py-2 border border-r border-control-border bg-gray-50 text-sm whitespace-pre-line rounded-l-[3px] font-mono overflow-auto">
              {grantStatement}
            </pre>
            <button
              type="button"
              className="flex items-center -ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light bg-gray-50 hover:bg-gray-100 focus:ring-control focus:outline-hidden focus-visible:ring-2 focus:ring-offset-1 rounded-r-[3px]"
              onClick={handleCopy}
              title="Copy"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
              </svg>
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
