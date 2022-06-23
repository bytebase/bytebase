<template>
  <div class="w-full flex flex-col justify-start" :class="props.className">
    <p
      class="w-full mt-1 text-sm text-gray-500"
      :class="props.createInstanceFlag ? 'max-w-xl' : ''"
    >
      {{
        props.dataSourceType === "ADMIN"
          ? $t("instance.sentence.create-admin-user")
          : $t("instance.sentence.create-readonly-user")
      }}
      <span
        v-if="!props.createInstanceFlag"
        class="normal-link select-none"
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
        v-if="props.engineType == 'MYSQL' || props.engineType == 'TIDB'"
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
          href="https://bytebase.com/docs/install/install-with-docker#start-a-local-mysql-server-for-testing?source=console"
          target="_blank"
          class="normal-link"
        >
          {{ $t("common.detailed-guide") }}
        </a>
      </template>
      <template v-else-if="props.engineType == 'CLICKHOUSE'">
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
      <template v-else-if="props.engineType == 'POSTGRES'">
        <BBAttention
          class="mb-1"
          :style="'WARN'"
          :title="$t('instance.sentence.create-user-example.postgres.warn')"
        />
        <i18n-t
          tag="p"
          keypath="instance.sentence.create-user-example.postgres.template"
        >
          <template #password>
            <span class="text-red-600">YOUR_DB_PWD</span>
          </template>
        </i18n-t>
      </template>
      <template v-else-if="props.engineType == 'SNOWFLAKE'">
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
      <div class="mt-2 flex flex-row">
        <span
          class="flex-1 min-w-0 w-full inline-flex items-center px-3 py-2 border border-r border-control-border bg-gray-50 sm:text-sm whitespace-pre"
        >
          {{ grantStatement(props.engineType, props.dataSourceType) }}
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
import { reactive, PropType } from "vue";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { useI18n } from "vue-i18n";
import { DataSourceType, EngineType } from "../types";
import { pushNotification } from "@/store";

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
  engineType: {
    type: String as PropType<EngineType>,
    default: "MYSQL",
  },
  dataSourceType: {
    type: String as PropType<DataSourceType>,
    default: "ADMIN",
  },
});

const { t } = useI18n();

const state = reactive<LocalState>({
  showCreateUserExample: props.createInstanceFlag,
});

const grantStatement = (
  engineType: EngineType,
  dataSourceType: DataSourceType
): string => {
  if (dataSourceType === "ADMIN") {
    switch (engineType) {
      case "MYSQL":
      case "TIDB":
        return "CREATE USER bytebase@'%' IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, \nFLUSH_TABLES, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE, \nREPLICATION_APPLIER, SESSION_VARIABLES_ADMIN \nON *.* to bytebase@'%';";
      case "CLICKHOUSE":
        return "CREATE USER bytebase IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT ALL ON *.* TO bytebase WITH GRANT OPTION;";
      case "SNOWFLAKE":
        return "CREATE OR REPLACE USER bytebase PASSWORD = 'YOUR_DB_PWD'\nDEFAULT_ROLE = \"ACCOUNTADMIN\"\nDEFAULT_WAREHOUSE = 'YOUR_COMPUTE_WAREHOUSE';\n\nGRANT ROLE \"ACCOUNTADMIN\" TO USER bytebase;";
      case "POSTGRES":
        return "CREATE USER bytebase WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';\n\nALTER USER bytebase WITH SUPERUSER;";
    }
  } else {
    switch (engineType) {
      case "MYSQL":
      case "TIDB":
        return "CREATE USER bytebase@'%' IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT SELECT, SHOW DATABASES, SHOW VIEW, USAGE ON *.* to bytebase@'%';";
      case "CLICKHOUSE":
        return "CREATE USER bytebase IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT SHOW TABLES, SELECT ON database.* TO bytebase;";
      case "SNOWFLAKE":
        return "CREATE OR REPLACE USER bytebase PASSWORD = 'YOUR_DB_PWD'\nDEFAULT_ROLE = \"ACCOUNTADMIN\"\nDEFAULT_WAREHOUSE = 'YOUR_COMPUTE_WAREHOUSE';\n\nGRANT ROLE \"ACCOUNTADMIN\" TO USER bytebase;";
      case "POSTGRES":
        return "CREATE USER bytebase WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';\n\nALTER USER bytebase WITH SUPERUSER;";
    }
  }
};

const toggleCreateUserExample = () => {
  state.showCreateUserExample = !state.showCreateUserExample;
};

const copyGrantStatement = () => {
  toClipboard(grantStatement(props.engineType, props.dataSourceType)).then(
    () => {
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("instance.copy-grant-statement"),
      });
    }
  );
};
</script>
