# Bytebase Frontend Style Guide

We are following [Bytebase's API Style Guide](https://github.com/bytebase/bytebase/blob/main/docs/api-style-guide.md) to ensure our style guide is **consistent**.

# URL naming

## Use slugs in URLs

Bytebase uses [slugs](https://en.wikipedia.org/wiki/Clean_URL#Slug) in URLs for resource identification. We use `/issue/hello-world-101` instead of `/issue/101`

_Rationale_: This makes URLs more clean and human-readable, especially when sharing via social media.

## Use lower case, kebab-case for phrases

Use `/anomaly-center` instead of `/anomalycenter` or `/anomalyCenter`

_Rationale_: Using `/anomalycenter` makes it more difficult to read. `/anomalyCenter` makes it more difficult to memoize and type because of mixed cases. Using `/anomaly-center` is good for readability and improves consistency with slugs.

## Use singular form even for collection resource

Use `/issue` instead of `/issues` to display the list of issues.

_Rationale_: Plural forms have several variations and it's hard for non-native English speakers to remember all the rules. And in practice, using singular form for collection resource won't cause confusion with the singular resource because they use different resource paths, e.g. `/issue` versus `/issue/:id`.

# Naming

## Naming variables and properties

We are following the "Property Name Convention" guide in [Bytebase's API Style Guide](https://github.com/bytebase/bytebase/blob/main/docs/api-style-guide.md#property-name-convention).

This allows us to follow vue recommended naming style such as [vue/attribute-hyphenation](https://eslint.vuejs.org/rules/attribute-hyphenation.html) and [vue/camelcase](https://eslint.vuejs.org/rules/camelcase.html). Making our code looks more likely to vue's "local accent".

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

# References

1. [Bytebase's API Style Guide](https://github.com/bytebase/bytebase/blob/main/docs/api-style-guide.md)
1. [Google's AIP](https://google.aip.dev/)
1. [Kubernetes API reference](https://kubernetes.io/docs/reference/)
