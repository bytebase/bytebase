<template>
  <NButton type="primary" @click="gotoSQLEditor">
    <template #icon>
      <heroicons-solid:terminal class="w-5 h-5" />
    </template>
    {{ $t("sql-editor.self") }}
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { useRouter } from "vue-router";
import { useIssueContext } from "@/components/IssueV1/logic";
import {
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
} from "@/router/sqlEditor";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import {
  extractInstanceResourceName,
  extractProjectResourceName,
} from "@/utils";
import { convertFromCELString } from "@/utils/issue/cel";

const router = useRouter();
const { issue } = useIssueContext();

const gotoSQLEditor = async () => {
  const grantRequest = issue.value.grantRequest!;
  const conditionExpression = await convertFromCELString(
    grantRequest.condition?.expression ?? ""
  );
  if (
    conditionExpression.databaseResources !== undefined &&
    conditionExpression.databaseResources.length > 0
  ) {
    const databaseResourceName =
      conditionExpression.databaseResources[0].databaseFullName;
    const db =
      await useDatabaseV1Store().getOrFetchDatabaseByName(databaseResourceName);
    if (isValidDatabaseName(db.name)) {
      const url = router.resolve({
        name: SQL_EDITOR_DATABASE_MODULE,
        params: {
          project: extractProjectResourceName(db.project),
          instance: extractInstanceResourceName(db.instance),
          database: db.databaseName,
        },
      });
      window.open(url.fullPath, "__BLANK");
      return;
    }
  }
  const url = router.resolve({
    name: SQL_EDITOR_HOME_MODULE,
  });
  window.open(url.fullPath, "__BLANK");
};
</script>
