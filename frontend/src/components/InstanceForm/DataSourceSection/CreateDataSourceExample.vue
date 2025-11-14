<template>
  <div
    v-if="grantStatement"
    class="w-full flex flex-col justify-start"
    :class="props.className"
  >
    <p class="w-full text-sm text-gray-500">
      <template v-if="isEngineUsingSQL">
        {{
          props.dataSourceType === DataSourceType.ADMIN
            ? $t("instance.sentence.create-admin-user")
            : $t("instance.sentence.create-readonly-user")
        }}
      </template>
      <template v-else>
        {{
          props.dataSourceType === DataSourceType.ADMIN
            ? $t("instance.sentence.create-admin-user-non-sql")
            : $t("instance.sentence.create-readonly-user-non-sql")
        }}
      </template>
      <span
        v-if="!props.createInstanceFlag"
        class="normal-link select-none ml-1"
        @click="toggleCreateUserExample"
      >
        {{ $t("instance.show-how-to-create") }}
      </span>
    </p>
    <div v-if="state.showCreateUserExample" class="text-sm text-main">
      <template
        v-if="
          props.engine === Engine.MYSQL ||
          props.engine === Engine.TIDB ||
          props.engine === Engine.OCEANBASE
        "
      >
        <i18n-t
          v-if="
            authenticationType ===
            DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
          "
          tag="span"
          keypath="instance.sentence.google-cloud-sql.mysql.template"
        >
          <template #user>
            <span class="font-semibold">
              {{ userName }}
            </span>
          </template>
        </i18n-t>
        <i18n-t
          v-else-if="
            authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
          "
          tag="span"
          keypath="instance.sentence.aws-rds.mysql.template"
        >
          <template #user>
            <span class="font-semibold">
              {{ userName }}
            </span>
          </template>
        </i18n-t>
        <i18n-t
          v-else
          tag="span"
          keypath="instance.sentence.create-user-example.mysql.template"
        >
          <template #user>
            <span class="font-semibold">
              {{ userName }}
            </span>
          </template>
          <template #password>
            <span class="text-red-600"> YOUR_DB_PWD </span>
          </template>
        </i18n-t>
      </template>
      <template v-else-if="props.engine === Engine.CLICKHOUSE">
        <i18n-t
          tag="p"
          keypath="instance.sentence.create-user-example.clickhouse.template"
        >
          <template #user>
            <span class="font-semibold">{{ userName }}</span>
          </template>
          <template #password>
            <span class="text-red-600">YOUR_DB_PWD</span>
          </template>
          <template #link>
            <a
              class="normal-link"
              href="https://clickhouse.com/docs/en/operations/access-rights/#access-control-usage"
              target="__blank"
            >
              {{
                $t(
                  "instance.sentence.create-user-example.clickhouse.sql-driven-workflow"
                )
              }}
            </a>
          </template>
        </i18n-t>
      </template>
      <template v-else-if="props.engine === Engine.POSTGRES">
        <BBAttention class="my-2" type="warning">
          {{ $t("instance.sentence.create-user-example.postgresql.warn") }}
        </BBAttention>
        <i18n-t
          v-if="
            authenticationType ===
            DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
          "
          tag="p"
          keypath="instance.sentence.google-cloud-sql.postgresql.template"
        >
          <template #user>
            <span class="font-semibold">
              {{ userName }}
            </span>
          </template>
        </i18n-t>
        <i18n-t
          v-else-if="
            authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
          "
          tag="p"
          keypath="instance.sentence.aws-rds.postgresql.template"
        >
          <template #user>
            <span class="font-semibold">
              {{ userName }}
            </span>
          </template>
        </i18n-t>
        <i18n-t
          v-else
          tag="p"
          keypath="instance.sentence.create-user-example.postgresql.template"
        >
          <template #user>
            <span class="font-semibold">{{ userName }}</span>
          </template>
          <template #password>
            <span class="text-red-600">YOUR_DB_PWD</span>
          </template>
        </i18n-t>
      </template>
      <template v-else-if="props.engine === Engine.SNOWFLAKE">
        <i18n-t
          tag="p"
          keypath="instance.sentence.create-user-example.snowflake.template"
        >
          <template #user>
            <span class="font-semibold">{{ userName }}</span>
          </template>
          <template #password>
            <span class="text-red-600">YOUR_DB_PWD</span>
          </template>
          <template #warehouse>
            <span class="text-red-600">YOUR_COMPUTE_WAREHOUSE</span>
          </template>
        </i18n-t>
      </template>
      <template v-else-if="props.engine === Engine.REDIS">
        <i18n-t
          tag="p"
          keypath="instance.sentence.create-user-example.redis.template"
        >
          <template #user>
            <span class="font-semibold">{{ userName }}</span>
          </template>
          <template #password>
            <span class="text-red-600"> YOUR_DB_PWD </span>
          </template>
        </i18n-t>
        <!-- TODO(xz): add a "detailed guide" link to docs here -->
      </template>
      <div class="mt-2 flex flex-row">
        <NConfigProvider
          class="flex-1 min-w-0 w-full inline-flex items-center px-3 py-2 border border-r border-control-border bg-gray-50 sm:text-sm whitespace-pre-line rounded-l-[3px]"
          :hljs="hljs"
        >
          <NCode
            word-wrap
            :language="languageOfEngineV1(engine)"
            :code="grantStatement"
          />
        </NConfigProvider>
        <div
          class="flex items-center -ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light disabled:text-gray-300 bg-gray-50 hover:bg-gray-100 disabled:bg-gray-50 focus:ring-control focus:outline-hidden focus-visible:ring-2 focus:ring-offset-1 disabled:cursor-not-allowed rounded-r-[3px]"
        >
          <CopyButton :content="grantStatement" />
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import hljs from "highlight.js/lib/core";
import { NCode, NConfigProvider } from "naive-ui";
import { computed, reactive } from "vue";
import { BBAttention } from "@/bbkit";
import { CopyButton } from "@/components/v2";
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

interface LocalState {
  showCreateUserExample: boolean;
}

const props = withDefaults(
  defineProps<{
    className?: string;
    createInstanceFlag: boolean;
    engine: Engine;
    dataSourceType: DataSourceType;
    authenticationType: DataSource_AuthenticationType;
  }>(),
  {
    className: "",
    createInstanceFlag: false,
    engine: Engine.MYSQL,
    dataSourceType: DataSourceType.ADMIN,
  }
);

const state = reactive<LocalState>({
  showCreateUserExample: props.createInstanceFlag,
});

const isEngineUsingSQL = computed(() => {
  return languageOfEngineV1(props.engine) === "sql";
});

const userName = computed(() => {
  return props.dataSourceType === DataSourceType.ADMIN
    ? DATASOURCE_ADMIN_USER_NAME
    : DATASOURCE_READONLY_USER_NAME;
});

const grantStatement = computed(() => {
  if (props.dataSourceType === DataSourceType.ADMIN) {
    const createUserStatement = `CREATE USER ${DATASOURCE_ADMIN_USER_NAME}@'%' IDENTIFIED BY 'YOUR_DB_PWD';`;
    switch (props.engine) {
      case Engine.MYSQL:
        if (
          props.authenticationType ===
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        ) {
          return `GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nRELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE \n/*!80000 , SET_USER_ID */\nON *.* to ${DATASOURCE_ADMIN_USER_NAME}@'%';`;
        } else if (
          props.authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
        ) {
          return `CREATE USER ${DATASOURCE_ADMIN_USER_NAME}@'%' IDENTIFIED WITH AWSAuthenticationPlugin AS 'RDS';\n\nALTER USER '${DATASOURCE_ADMIN_USER_NAME}'@'%' REQUIRE SSL;\n\nGRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nRELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE \n/*!80000 , SET_USER_ID */\nON *.* to ${DATASOURCE_ADMIN_USER_NAME}@'%';`;
        }
        // RELOAD, LOCK TABLES: enables use of explicit LOCK TABLES statements for backups.
        // REPLICATION CLIENT: enables use of the SHOW MASTER STATUS, SHOW SLAVE STATUS, and SHOW BINARY LOGS statements.
        // REPLICATION SLAVE: use of the SHOW SLAVE HOSTS, SHOW RELAYLOG EVENTS, and SHOW BINLOG EVENTS statements. This privilege is also required to use the mysqlbinlog options --read-from-remote-server (-R) and --read-from-remote-master.
        // REPLICATION_APPLIER: execute the internal-use BINLOG statements used by mysqlbinlog.
        // SESSION_VARIABLES_ADMIN: use of the SET sql_log_bin statements during PITR.
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
        return `-- Option 1: grant ACCOUNTADMIN role

CREATE OR REPLACE USER ${DATASOURCE_ADMIN_USER_NAME} PASSWORD = 'YOUR_DB_PWD'
DEFAULT_ROLE = "ACCOUNTADMIN"
DEFAULT_WAREHOUSE = 'YOUR_COMPUTE_WAREHOUSE';

GRANT ROLE "ACCOUNTADMIN" TO USER ${DATASOURCE_ADMIN_USER_NAME};

-- Option 2: grant more granular privileges

CREATE OR REPLACE ROLE BYTEBASE;

-- If using non-enterprise edition, the following commands may encounter error likes 'Unsupported feature GRANT/REVOKE APPLY TAG', you can skip those unsupported GRANTs.
-- Grant the least privileges required by Bytebase 

GRANT CREATE DATABASE, EXECUTE TASK, IMPORT SHARE, APPLY MASKING POLICY, APPLY ROW ACCESS POLICY, APPLY TAG ON ACCOUNT TO ROLE BYTEBASE;

GRANT IMPORTED PRIVILEGES ON DATABASE SNOWFLAKE TO ROLE BYTEBASE;

GRANT USAGE ON WAREHOUSE "YOUR_COMPUTE_WAREHOUSE" TO ROLE BYTEBASE;

CREATE OR REPLACE USER BYTEBASE
  PASSWORD = 'YOUR_PWD'
  DEFAULT_ROLE = "BYTEBASE"
  DEFAULT_WAREHOUSE = "YOUR_COMPUTE_WAREHOUSE";

GRANT ROLE "BYTEBASE" TO USER BYTEBASE;

GRANT ROLE "BYTEBASE" TO ROLE SYSADMIN;

-- For each database to be managed by Bytebase, you need to grant the following privileges

GRANT ALL PRIVILEGES ON DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL SCHEMAS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE SCHEMAS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

-- Then, grant the required schema object privileges on the database to the Bytebase role
GRANT ALL PRIVILEGES ON ALL EVENT TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE EVENT TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL EXTERNAL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE EXTERNAL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL FILE FORMATS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE FILE FORMATS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE FUNCTIONS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL MATERIALIZED VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE MATERIALIZED VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL PROCEDURES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE PROCEDURES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL SEQUENCES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE SEQUENCES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL STAGES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE STAGES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL STREAMS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE STREAMS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL TASKS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE TASKS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

-- ALERT and POLICY are the features in Snowflake Enterprise Edition, you can skip those GRANTs if you are using standard edition.
GRANT ALL PRIVILEGES ON ALL ALERTS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE ALERTS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL PASSWORD POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE PASSWORD POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL ROW ACCESS POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE ROW ACCESS POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL MASKING POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE MASKING POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL SESSION POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE SESSION POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

-- PIPE are not allowed to be bulk granted, you need to grant them one by one.
GRANT ALL PRIVILEGES ON PIPE {{PIPE_NAME}} IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
`;
      case Engine.POSTGRES:
        if (
          props.authenticationType ===
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        ) {
          return `GRANT pg_write_all_data TO "${DATASOURCE_ADMIN_USER_NAME}@{project-id}.iam";

-- If you need to create databases via Bytebase
ALTER USER "${DATASOURCE_ADMIN_USER_NAME}@{project-id}.iam" WITH CREATEDB;`;
        } else if (
          props.authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
        ) {
          return `CREATE USER ${DATASOURCE_ADMIN_USER_NAME};

GRANT rds_iam TO ${DATASOURCE_ADMIN_USER_NAME};

GRANT pg_write_all_data TO ${DATASOURCE_ADMIN_USER_NAME};

-- If you need to create databases via Bytebase
ALTER USER ${DATASOURCE_ADMIN_USER_NAME} WITH CREATEDB;`;
        }
        return `CREATE USER ${DATASOURCE_ADMIN_USER_NAME} WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';\n\nALTER USER ${DATASOURCE_ADMIN_USER_NAME} WITH SUPERUSER;`;
      case Engine.REDSHIFT:
        return `CREATE USER ${DATASOURCE_ADMIN_USER_NAME} WITH PASSWORD 'YOUR_DB_PWD' CREATEUSER CREATEDB;`;
      case Engine.MONGODB:
        return `use admin;
db.createUser({
  user: "${DATASOURCE_ADMIN_USER_NAME}",
  pwd: "YOUR_DB_PWD",
  roles: [
    {role: "readWriteAnyDatabase", db: "admin"},
    {role: "dbAdminAnyDatabase", db: "admin"},
    {role: "userAdminAnyDatabase", db: "admin"}
  ]
});
`;
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
        return `-- If you use Cloud RDS, you need to checkout their documentation for setting up a semi-super privileged user.
CREATE LOGIN ${DATASOURCE_ADMIN_USER_NAME} WITH PASSWORD = 'YOUR_DB_PWD';
ALTER SERVER ROLE sysadmin ADD MEMBER ${DATASOURCE_ADMIN_USER_NAME};`;
      case Engine.ORACLE:
        return `-- If you use Cloud RDS, you need to checkout their documentation for setting up a semi-super privileged user.
CREATE USER ${DATASOURCE_ADMIN_USER_NAME} IDENTIFIED BY 'YOUR_DB_PWD';
GRANT ALL PRIVILEGES TO ${DATASOURCE_ADMIN_USER_NAME};`;
    }
  } else {
    const mysqlReadonlyStatement = `CREATE USER ${DATASOURCE_READONLY_USER_NAME}@'%' IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT SELECT, SHOW DATABASES, SHOW VIEW, USAGE ON *.* to ${DATASOURCE_READONLY_USER_NAME}@'%';`;
    switch (props.engine) {
      case Engine.MYSQL:
        if (
          props.authenticationType ===
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        ) {
          return `GRANT SELECT, SHOW DATABASES, SHOW VIEW, USAGE ON *.* to ${DATASOURCE_READONLY_USER_NAME}@'%';`;
        } else if (
          props.authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
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
        return `-- Option 1: grant ACCOUNTADMIN role

CREATE OR REPLACE USER ${DATASOURCE_READONLY_USER_NAME} PASSWORD = 'YOUR_DB_PWD'
DEFAULT_ROLE = "ACCOUNTADMIN"
DEFAULT_WAREHOUSE = 'YOUR_COMPUTE_WAREHOUSE';

GRANT ROLE "ACCOUNTADMIN" TO USER ${DATASOURCE_READONLY_USER_NAME};

-- Option 2: grant more granular privileges

CREATE OR REPLACE ROLE BYTEBASE_READER;

-- If using non-enterprise edition, the following commands may encounter error likes 'Unsupported feature GRANT/REVOKE APPLY TAG', you can skip those unsupported GRANTs.
-- Grant the least privileges required by Bytebase 

GRANT IMPORT SHARE TO ROLE BYTEBASE_READER;

GRANT IMPORTED PRIVILEGES ON DATABASE SNOWFLAKE TO ROLE BYTEBASE_READER;

GRANT USAGE ON WAREHOUSE "YOUR_COMPUTE_WAREHOUSE" TO ROLE BYTEBASE_READER;

CREATE OR REPLACE USER BYTEBASE_READER
  PASSWORD = 'YOUR_PWD'
  DEFAULT_ROLE = "BYTEBASE_READER"
  DEFAULT_WAREHOUSE = "YOUR_COMPUTE_WAREHOUSE";

GRANT ROLE "BYTEBASE_READER" TO USER BYTEBASE_READER;

GRANT ROLE "BYTEBASE_READER" TO ROLE SYSADMIN;

-- For each database to be managed by Bytebase, you need to grant the following privileges

GRANT ALL PRIVILEGES ON DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL SCHEMAS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE SCHEMAS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

-- Then, grant the required schema object privileges on the database to the Bytebase role
GRANT ALL PRIVILEGES ON ALL EVENT TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE EVENT TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL EXTERNAL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE EXTERNAL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL FILE FORMATS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE FILE FORMATS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE FUNCTIONS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL MATERIALIZED VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE MATERIALIZED VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL PROCEDURES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE PROCEDURES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL SEQUENCES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE SEQUENCES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL STAGES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE STAGES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL STREAMS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE STREAMS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE TABLES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL TASKS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE TASKS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE VIEWS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

-- ALERT and POLICY are the features in Snowflake Enterprise Edition, you can skip those GRANTs if you are using standard edition.
GRANT ALL PRIVILEGES ON ALL ALERTS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE ALERTS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL PASSWORD POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE PASSWORD POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL ROW ACCESS POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE ROW ACCESS POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL MASKING POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE MASKING POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL SESSION POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE SESSION POLICIES IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

-- PIPE are not allowed to be bulk granted, you need to grant them one by one.
GRANT ALL PRIVILEGES ON PIPE {{PIPE_NAME}} IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
`;
      case Engine.POSTGRES:
        if (
          props.authenticationType ===
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        ) {
          return `GRANT pg_read_all_data TO "${DATASOURCE_READONLY_USER_NAME}@{project-id}.iam";`;
        } else if (
          props.authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM
        ) {
          return `CREATE USER ${DATASOURCE_READONLY_USER_NAME};\n\nGRANT rds_iam TO ${DATASOURCE_READONLY_USER_NAME};\n\nGRANT pg_read_all_data TO ${DATASOURCE_READONLY_USER_NAME};`;
        }
        return `CREATE USER ${DATASOURCE_READONLY_USER_NAME} WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';\n\nGRANT pg_read_all_data TO ${DATASOURCE_READONLY_USER_NAME};`;
      case Engine.MONGODB:
        return `use admin;
db.createUser({
  user: "${DATASOURCE_READONLY_USER_NAME}",
  pwd: "YOUR_DB_PWD",
  roles: [
    {role: "readAnyDatabase", db: "admin"}
  ]
});
        `;
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
  return ""; // fallback
});

const toggleCreateUserExample = () => {
  state.showCreateUserExample = !state.showCreateUserExample;
};
</script>
