# Naming

## Variables and properties

We are following the "Property Name Convention" guide in [Bytebase's API Style Guide](https://github.com/bytebase/bytebase/blob/main/docs/api-style-guide.md#property-name-convention).

This allows us to follow vue recommended naming style such as [vue/attribute-hyphenation](https://eslint.vuejs.org/rules/attribute-hyphenation.html) and [vue/camelcase](https://eslint.vuejs.org/rules/camelcase.html). Making our code looks more likely to vue's "local accent".

## Files and directories

We use different naming style to different types of files and directories.

- Naming components and views with PascalCase. Component directories also follow this rule. e.g., `DatabaseOverviewPanel.vue`, `DatabaseDetail.vue`, `ActivityTable/ActivityCommentLink.vue`.
- If a file's default export is a class, use PascalCase, too. e.g., `DatabaseSchemaUpdateTemplate.ts`.
- Naming composable function files with camelCase prefixed by "use". e.g., `composables/useSQLEditorConnection.ts`.
- Naming other files and directories with lower case kebab-case. e.g., `data-source-type.ts`, `fe-style-guide.md`, `store/mutation-types.ts`.

# Vue components

## Composition API

We write vue components with [Composition API](https://vuejs.org/guide/extras/composition-api-faq.html). It's easier and more practical to extract reusable logic as composable functions than mixins.

We also recommend to use it together with the `<script setup>` syntax in Single-File Components.

## Templates

We prefer templates rather than JSX. Since templates are better optimized in compile stage of Vue 3. It also helps us to simplify the compile and build toolchain. And we benefits from Vue Single-File Components.

## Component local state pattern

We recommend using a "local state" pattern when components mutate their properties or provide a `v-model` property. This also helps us to avoid complaints from [vue/no-mutating-props](https://eslint.vuejs.org/rules/no-mutating-props.html).

See [BBSwitch](https://github.com/bytebase/bytebase/blob/main/frontend/src/bbkit/BBSwitch.vue) as an example of this pattern.

# Linting and formatting

## Principles

We are using Vue, TypeScript and Tailwind CSS. Following their recommended style guide improves consistency. Especially when introducing new 3rd-party dependencies.

## Tools

We are using [ESLint](https://eslint.org/) and [Prettier](https://prettier.io/) as our lint and format tools. [plugin:vue/vue3-recommended](https://eslint.vuejs.org/) as our default lint rules.

See [the configuration file](https://github.com/bytebase/bytebase/blob/main/frontend/.eslintrc.js) to learn more about the rules.
