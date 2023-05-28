# I18n Guide

Bytebase supports English and Chinese. We provide the language toggle on the Sign-up/Sign-in page and the user profile dropdown menu on the top-right of the dashboard.

## How do we translate

### Overview

Required tools.

* `vue-i18n` for vue project.
* `i18n-ally` for VSCode Extension.

With these tools, we could extract translations from code without opening the [i18n assets](#i18n-assets).

### Vue I18n

In our frontend project, we use `vue-i18n` to translate all the messages. Vue I18n is an internationalization plugin for Vue.js.

Learn about [vue-i18n](https://vue-i18n.intlify.dev/)

### i18n-ally

We use a VSCode extension called `i18n-ally`. It provides a simple and seamless translation workflow.

Some usefully features:

* Extract Translations from Code
* Inline Annotations
* Hover and Direct Actions
* Mangae All Translations in One Place
* Editor UI & Review System
* Report Missing Translations
* Machine Translation

Learn about [VSCode Extension i18n-ally](https://marketplace.visualstudio.com/items?itemName=lokalise.i18n-ally)

### i18n assets

The `i18n assets` refers to the locales files in the `bytebase/frontend/src/locales` dir.

`en.yml` is for English and `zh-CN.yml` is for Chinese.

### Translate message in `<template>`

Most translation work would be done in the Vue file under the `<template>` tag which is the page template. vue-i18n injects a Vue global variable `$t` globally. So that we can translate directly from template using the `$t` function.

For example:

```html
{{ $t("quick-start.self") }}
```

### Translate message in `<script>`

Sometimes, we write some texts as constants, such as fixed drop-down menu options. The translation of texts in the `<script>` tag is slightly different, and we need to use the Vue i18n API.

For example:

```ts
import { useI18n } from "vue-i18n";

export default {
  ...,
  setup () {
    const { t } = useI18n()

    return {
      quickStart: t("quick-start.self")
    }
  },
  ...
}
```

### Translate message in `.ts` file

Sometimes, some constants may be referenced by multiple files, and we extract those constants into a separate `.ts` file. At this time, we export the `t` function from `plugins/i18n.ts`.

```ts
import { t } from "../plugins/i18n";
export const PROJECT_HOOK_TYPE_ITEM_LIST: () => ProjectWebhookTypeIte[] =
  () => [
     ...,
    {
      type: "bb.plugin.webhook.discord",
      name: t("common.discord"),
      urlPrefix: "https://discord.com/api/webhooks",
    },
    ...
  ];
```

For more advanced usage, check out the [vue-i18n] documents.

## Contributes

If you would like to help a language's translation up to date, please follow this guide.

### How to start

#### Clone to local

```bash
npx degit bytebase/bytebase bytebase
cd bytebase/frontend
pnpm i && pnpm dev
```

#### Extract some messages

Follow the [How do we translate?](#how-do-we-translate) guide.

## Tools

Some useful frontend dev tools for i18n in Vue Project.

* [vue-i18n](https://vue-i18n.intlify.dev/)
* [VSCode Extension i18n-ally](https://marketplace.visualstudio.com/items?itemName=lokalise.i18n-ally) - 🌍 All in one i18n extension for VS Code
