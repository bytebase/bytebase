<template>
  <div class="flex flex-col gap-y-2 text-sm">
    <div class="flex flex-col gap-y-2 p-2">
      <div class="flex items-center gap-x-3">
        <DatabaseSelect v-model:database="databaseUID" :clearable="true" />
        <div>{{ database.name }}</div>
      </div>
    </div>
    <div v-if="component === 'RAW'" class="w-full h-auto px-2">
      <MonacoTextModelEditor :model="model" v-bind="editorProps" />
    </div>

    <div v-if="component === 'WRAPPED'" class="w-full h-auto px-2">
      <MonacoEditor
        :filename="filename"
        :content="content"
        :language="language"
        v-bind="editorProps"
        @update:content="onUpdateContent"
      />
    </div>

    <div class="flex flex-col gap-y-2 p-2">
      <NRadioGroup v-if="false" v-model:value="component">
        <NRadio value="RAW">MonacoTextModelEditor</NRadio>
        <NRadio value="WRAPPED">MonacoEditor</NRadio>
      </NRadioGroup>
      <NRadioGroup v-if="false" v-model:value="language">
        <NRadio value="sql">SQL</NRadio>
        <NRadio value="javascript">JS</NRadio>
        <NRadio value="redis">REDIS</NRadio>
      </NRadioGroup>
      <NCheckbox v-model:checked="readonly">Readonly</NCheckbox>

      <NInput
        v-if="false"
        v-model:value="content"
        type="textarea"
        class="font-mono"
        :autosize="{
          minRows: 3,
          maxRows: 8,
        }"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NCheckbox, NInput, NRadio, NRadioGroup } from "naive-ui";
import { ref } from "vue";
import { computed } from "vue";
import {
  MonacoEditor,
  MonacoTextModelEditor,
  useMonacoTextModel,
} from "@/components/MonacoEditor";
import { useDatabaseV1ByUID, useDatabaseV1Store } from "@/store";
import { Language } from "@/types";
import { DatabaseSelect } from "../v2";

const component = ref<"RAW" | "WRAPPED">("WRAPPED");
const language = ref<Language>("sql");
const readonly = ref(false);
const databaseUID = ref(head(useDatabaseV1Store().databaseList)?.uid);
const { database } = useDatabaseV1ByUID(
  computed(() => databaseUID.value ?? "-1")
);

const editorProps = computed(() => ({
  readonly: readonly.value,
  autoCompleteContext: {
    instance: database.value.instance,
    database: database.value.name,
  },
  autoHeight: {
    min: 40,
    max: 240,
  },
  class: "w-full h-full border",
}));

const contents = {
  sql: ref(
    `SELECT
  dept_name,
  COUNT(gender) AS gender_count,
  SUM(IF(gender = 'M', 1, 0)) AS male_count,
  SUM(IF(gender = 'F', 1, 0)) AS female_count,
  SUM(IF(gender = 'M', 1, 0)) / COUNT(gender) AS male_ratio,
  SUM(IF(gender = 'F', 1, 0)) / COUNT(gender) AS female_ratio
FROM
  employee
  JOIN dept_emp ON employee.emp_no = dept_emp.emp_no
  JOIN department ON dept_emp.dept_no = department.dept_no
GROUP BY
  dept_name`
  ),
  javascript: ref(`const main = async () => {
  console.log("hello world")
}`),
  redis: ref(`SET foo bar
GET foo
DEL foo
EXISTS foo`),
};
const models = {
  sql: useMonacoTextModel("test.sql", contents.sql, "sql"),
  javascript: useMonacoTextModel("test.js", contents.javascript, "javascript"),
  redis: useMonacoTextModel("test.redis", contents.redis, "redis"),
};

const model = computed(() => {
  return models[language.value]?.value;
});
const filename = computed(() => {
  return `v2.${language.value.toLowerCase()}`;
});
const content = computed(() => {
  return contents[language.value]?.value ?? "";
});

const onUpdateContent = (content: string) => {
  const contentRef = contents[language.value];
  if (!contentRef) return;
  contentRef.value = content;
};
</script>
