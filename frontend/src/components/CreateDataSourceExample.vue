<template>
  <div class="w-full flex flex-col justify-start" :class="props.className">
    <p class="w-full mt-1 text-sm text-gray-500">
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
    <!-- Specify the fixed width so the create instance dialog width won't shift when switching engine types-->
    <div
      v-if="state.showCreateUserExample"
      class="mt-2 text-sm text-main w-208"
    >
      <template
        v-if="
          props.engine === Engine.MYSQL ||
          props.engine === Engine.TIDB ||
          props.engine === Engine.OCEANBASE ||
          props.engine === Engine.RISINGWAVE
        "
      >
        <i18n-t
          tag="p"
          keypath="instance.sentence.create-user-example.mysql.template"
        >
          <template #user>{{
            $t("instance.sentence.create-user-example.mysql.user")
          }}</template>
          <template #password>
            <span class="text-red-600">
              {{ $t("instance.sentence.create-user-example.mysql.password") }}
            </span>
          </template>
        </i18n-t>
        <a
          href="https://www.bytebase.com/docs/get-started/install/local-mysql-instance?source=console"
          target="_blank"
          class="normal-link"
        >
          {{ $t("common.detailed-guide") }}
        </a>
      </template>
      <template v-else-if="props.engine === Engine.CLICKHOUSE">
        <i18n-t
          tag="p"
          keypath="instance.sentence.create-user-example.clickhouse.template"
        >
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
        <BBAttention
          class="mb-1"
          :style="'WARN'"
          :title="$t('instance.sentence.create-user-example.postgresql.warn')"
        />
        <i18n-t
          tag="p"
          keypath="instance.sentence.create-user-example.postgresql.template"
        >
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
          <template #user>{{
            $t("instance.sentence.create-user-example.redis.user")
          }}</template>
          <template #password>
            <span class="text-red-600">
              {{ $t("instance.sentence.create-user-example.redis.password") }}
            </span>
          </template>
        </i18n-t>
        <!-- TODO(xz): add a "detailed guide" link to docs here -->
      </template>
      <div class="mt-2 flex flex-row">
        <span
          class="flex-1 min-w-0 w-full inline-flex items-center px-3 py-2 border border-r border-control-border bg-gray-50 sm:text-sm whitespace-pre-line"
        >
          {{ grantStatement(props.engine, props.dataSourceType) }}
        </span>
        <button
          tabindex="-1"
          class="-ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light disabled:text-gray-300 bg-gray-50 hover:bg-gray-100 disabled:bg-gray-50 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1 disabled:cursor-not-allowed"
          @click.prevent="copyGrantStatement"
        >
          <heroicons-outline:clipboard class="w-6 h-6" />
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { reactive, PropType, computed } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { languageOfEngineV1 } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { DataSourceType } from "@/types/proto/v1/instance_service";
import { engineNameV1 } from "@/utils";

interface LocalState {
  showCreateUserExample: boolean;
}

const props = defineProps({
  className: {
    type: String,
    default: "",
  },
  createInstanceFlag: {
    type: Boolean,
    default: false,
  },
  engine: {
    type: Number as PropType<Engine>,
    default: Engine.MYSQL,
  },
  dataSourceType: {
    type: Number as PropType<DataSourceType>,
    default: DataSourceType.ADMIN,
  },
});

const { t } = useI18n();

const state = reactive<LocalState>({
  showCreateUserExample: props.createInstanceFlag,
});

const isEngineUsingSQL = computed(() => {
  return languageOfEngineV1(props.engine) === "sql";
});

const grantStatement = (
  engine: Engine,
  dataSourceType: DataSourceType
): string => {
  if (dataSourceType === DataSourceType.ADMIN) {
    switch (engine) {
      case Engine.MYSQL:
        // RELOAD, LOCK TABLES: enables use of explicit LOCK TABLES statements for backups.
        // REPLICATION CLIENT: enables use of the SHOW MASTER STATUS, SHOW SLAVE STATUS, and SHOW BINARY LOGS statements.
        // REPLICATION SLAVE: use of the SHOW SLAVE HOSTS, SHOW RELAYLOG EVENTS, and SHOW BINLOG EVENTS statements. This privilege is also required to use the mysqlbinlog options --read-from-remote-server (-R) and --read-from-remote-master.
        // REPLICATION_APPLIER: execute the internal-use BINLOG statements used by mysqlbinlog.
        // SESSION_VARIABLES_ADMIN: use of the SET sql_log_bin statements during PITR.
        return "CREATE USER bytebase@'%' IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nRELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE \n/*!80000 , REPLICATION_APPLIER, SYSTEM_VARIABLES_ADMIN, SET_USER_ID */\nON *.* to bytebase@'%';";
      case Engine.TIDB:
        return "CREATE USER bytebase@'%' IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nLOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE \nON *.* to bytebase@'%';";
      case Engine.MARIADB:
        return "CREATE USER bytebase@'%' IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nRELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE \nON *.* to bytebase@'%';";
      case Engine.OCEANBASE:
        return "CREATE USER bytebase@'%' IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT ALTER, CREATE, CREATE VIEW, DELETE, DROP, INDEX, INSERT, \nPROCESS, SELECT, SHOW DATABASES, SHOW VIEW, UPDATE, USAGE, \nREPLICATION CLIENT, REPLICATION SLAVE \nON *.* to bytebase@'%';";
      case Engine.CLICKHOUSE:
        return "CREATE USER bytebase IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT ALL ON *.* TO bytebase WITH GRANT OPTION;";
      case Engine.SNOWFLAKE:
        return `-- Option 1: grant ACCOUNTADMIN role

CREATE OR REPLACE USER bytebase PASSWORD = 'YOUR_DB_PWD'
DEFAULT_ROLE = "ACCOUNTADMIN"
DEFAULT_WAREHOUSE = 'YOUR_COMPUTE_WAREHOUSE';

GRANT ROLE "ACCOUNTADMIN" TO USER bytebase;

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

GRANT ALL PRIVILEGES ON ALL PASSWORD POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE PASSWORD POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL ROW ACCESS POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE ROW ACCESS POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL MASKING POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE MASKING POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

GRANT ALL PRIVILEGES ON ALL SESSION POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
GRANT ALL PRIVILEGES ON FUTURE SESSION POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;

-- PIPE are not allowed to be bulk granted, you need to grant them one by one.
GRANT ALL PRIVILEGES ON PIPE {{PIPE_NAME}} IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE;
`;
      case (Engine.POSTGRES, Engine.RISINGWAVE):
        return "CREATE USER bytebase WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';\n\nALTER USER bytebase WITH SUPERUSER;";
      case Engine.REDSHIFT:
        return "CREATE USER bytebase WITH PASSWORD 'YOUR_DB_PWD' CREATEUSER CREATEDB;";
      case Engine.MONGODB:
        return 'use admin;\ndb.createUser({\n\tuser: "bytebase", \n\tpwd: "YOUR_DB_PWD", \n\troles: [\n\t\t{role: "readWriteAnyDatabase", db: "admin"},\n\t\t{role: "dbAdminAnyDatabase", db: "admin"},\n\t\t{role: "userAdminAnyDatabase", db: "admin"}\n\t]\n});';
      case Engine.SPANNER:
        return "";
      case Engine.REDIS:
        return "ACL SETUSER bytebase on >YOUR_DB_PWD +@all &*";
      case Engine.MSSQL:
        return "-- If you use Cloud RDS, you need to checkout their documentation for setting up a semi-super privileged user.\n\nCREATE LOGIN bytebase WITH PASSWORD = 'YOUR_DB_PWD';\nALTER SERVER ROLE sysadmin ADD MEMBER bytebase;";
      case Engine.ORACLE:
        return "-- If you use Cloud RDS, you need to checkout their documentation for setting up a semi-super privileged user.\n\nCREATE USER bytebase IDENTIFIED BY 'YOUR_DB_PWD';\nGRANT ALL PRIVILEGES TO bytebase;";
      case Engine.DM:
        return 'CREATE USER BYTEBASE IDENTIFIED BY "YOUR_DB_PWD";\nGRANT "DBA" TO BYTEBASE;';
    }
  } else {
    switch (engine) {
      case Engine.MYSQL:
      case Engine.TIDB:
      case Engine.OCEANBASE:
        return "CREATE USER bytebase@'%' IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT SELECT, SHOW DATABASES, SHOW VIEW, USAGE ON *.* to bytebase@'%';";
      case Engine.CLICKHOUSE:
        return "CREATE USER bytebase IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT SHOW TABLES, SELECT ON database.* TO bytebase;";
      case Engine.SNOWFLAKE:
        return `-- Option 1: grant ACCOUNTADMIN role

CREATE OR REPLACE USER bytebase PASSWORD = 'YOUR_DB_PWD'
DEFAULT_ROLE = "ACCOUNTADMIN"
DEFAULT_WAREHOUSE = 'YOUR_COMPUTE_WAREHOUSE';

GRANT ROLE "ACCOUNTADMIN" TO USER bytebase;

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

GRANT ALL PRIVILEGES ON ALL PASSWORD POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE PASSWORD POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL ROW ACCESS POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE ROW ACCESS POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL MASKING POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE MASKING POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

GRANT ALL PRIVILEGES ON ALL SESSION POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
GRANT ALL PRIVILEGES ON FUTURE SESSION POLICYS IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;

-- PIPE are not allowed to be bulk granted, you need to grant them one by one.
GRANT ALL PRIVILEGES ON PIPE {{PIPE_NAME}} IN DATABASE {{YOUR_DB_NAME}} TO ROLE BYTEBASE_READER;
`;
      case (Engine.POSTGRES, Engine.RISINGWAVE):
        return "CREATE USER bytebase WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';\n\nALTER USER bytebase WITH SUPERUSER;";
      case Engine.MONGODB:
        return 'use admin;\ndb.createUser({\n\tuser: "bytebase", \n\tpwd: "YOUR_DB_PWD", \n\troles: [\n\t\t{role: "readAnyDatabase", db: "admin"},\n\t\t{role: "dbAdminAnyDatabase", db: "admin"},\n\t\t{role: "userAdminAnyDatabase", db: "admin"}\n\t]\n});';
      case Engine.SPANNER:
        return "";
      case Engine.REDIS:
        return "ACL SETUSER bytebase on >YOUR_DB_PWD +@read &*";
    }
  }
  return ""; // fallback
};

const toggleCreateUserExample = () => {
  state.showCreateUserExample = !state.showCreateUserExample;
};

const copyGrantStatement = () => {
  toClipboard(grantStatement(props.engine, props.dataSourceType)).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("instance.copy-grant-statement", {
        engine: engineNameV1(props.engine),
      }),
    });
  });
};
</script>
