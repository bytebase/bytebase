# I18n Guide

Bytebase now both supports English and Chinese languages. We provide the language toggle UI on the Sign-up/Sign-in page and on the user profile dropdown menu on the top-right of the dashboard.

## How do we translate?

### Overview

Some tools we required.

* `vue-i18n` for vue project.
* `i18n-ally` for VSCode Extension.

With these tools. We could extract translations from code without open the [i18n assets](#i18n-assets).

### Vue I18n

In our frontend project. We use `vue-i18n` to translate all the messages. Vue I18n is internationalization plugin for Vue.js.

Learn about [vue-i18n](https://vue-i18n.intlify.dev/)

### i18n-ally

We use a VSCode extension called `i18n-ally` to improve our translation progress. It provide a simple and seemless workflow of translations.

Some usefully features about it:

* Extract Translations from Code
* Inline Annotations
* Hover and Direct Actions
* Mangae All Translations in One Place
* Editor UI & Review System
* Report Missing Translations
* Machine Translation

Learn about [VSCode Extension i18n-ally](https://marketplace.visualstudio.com/items?itemName=lokalise.i18n-ally)

### i18n assets

The `i18n assets` was the locales files in the `bytebase/frontend/src/locales` dir.

We defined the `en.yml` file presented English messages and the `zh-CN.yml` file presented Chinese messages.

Our team will maintain the i18n assets for a long time.

### Translate message in `<template>`

Most of the messages translation works should be done in the Vue file under the `<tempalte>` tag. This part is the page template. vue-i18n injects a Vue global variable `$t` globally. So that we can translate directly from template using the `$t` function.

For example:

```html
{{ $t("common.quickstart") }}
```

### Translate message in `<script>`

Sometimes, we write some texts as constants, such as fixed drop-down menu options. The translation of texts in the `<script>` tag is slightly different, and we need to use the Vue i18n API to complete.

For example:

```ts
import { useI18n } from "vue-i18n";

export default {
  ...,
  setup () {
    const { t } = useI18n()
    
    return {
      quickStart: t("common.quickstart")
    }
  },
  ...
}
```

### Translate message in `.ts` file

Sometimes, some constants may be referenced by multiple files, so this part of the code is moved to a separate `.ts` file. At this time, we export the `t` function from `plugins/i18n.ts`.

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

For more skills of doing translations. check out the [vue-i18n] documents.

## Contributes

If you would like to help a language's translation up to date. Please follow this guide.

### Suggestion or Improvement

#### Clone to local

```bash
npx degit bytebase/bytebase bytebase
cd bytebase/frontend
yarn && yarn dev
```

#### Extract some messages

Follow the [How do we translate?](#how-do-we-translate) guide.

### Add another language

If you want to add another language. You can create your language YAML file in the i18n assets dir(bytebase/frontend/src/locales) and translate the message to align with `en.yml`

## Tools

Some usefully frontend dev tools for i18n in Vue Project.

* [vue-i18n](https://vue-i18n.intlify.dev/)
* [VSCode Extension i18n-ally](https://marketplace.visualstudio.com/items?itemName=lokalise.i18n-ally) - 🌍 All in one i18n extension for VS Code
