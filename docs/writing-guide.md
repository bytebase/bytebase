# Writing Guide

We produce content in both English and Chinese. We need to pay attention to the formatting in order to deliver a good reading experience.

# Pure English Article

Use English letters and 半角 symbols.

# Pure Chinese Article

Use Chinese characters and 全角 symbols, pure Chinese article is rare since many terms are English only.

# Mixed English and Chinese

Most aritcles fall into this category. The formatting gets tricky because we need to choose between 半角 and 全角. In general, we follow [中文文案排版指北](https://github.com/sparanoid/chinese-copywriting-guidelines/blob/master/README.zh-Hans.md), with the following exceptions:

1. For parentheses, use `半角英文括号()` instead of `全角括号（）`.

# Example

> ❌ Github是一个代码版本控制系统（Version Control System）,它的吉祥物是一只章鱼猫，英文叫 “Octocat”.
>
> ✅ GitHub 是一个代码版本控制系统 (Version Control System)，它的吉祥物是一只章鱼猫，英文叫「Octocat」。

Let's go over the issues one by one:

1. Case. Respect other brands.
   > ❌ `Github`
   >
   > ✅ `GitHub`
1. Single whitespace between English and Chinese
   > ❌ `GitHub是一个`
   >
   > ✅ `GitHub 是一个`
1. Parentheses
   > ❌ `（Version Control System）`
   >
   > ✅ `(Version Control System)`
1. Comma
   > ❌ `,`
   >
   > ✅ `，`
1. Quotation mark
   > ❌ `“Octocat”`
   >
   > ✅ `「Octocat」`
1. Period
   > ❌ `.`
   >
   > ✅ `。`

# Technical Writing Style Guide

We mainly follow [Google's style guide](https://developers.google.com/style)

1. Bold for UI Label. Skip 'button','label' and etc if it's obvious.

> ✅ Click **Settings**
>
> ❌ Click **Settings** button
>
> ❌ Click “Settings”
>
> ❌ Click `Settings`
>
> ❌ Click Settings button


2. Bold and initial capitalized for concept, roles, and etc. 

> ✅ **Project** is the container to group logically related **Databases**, **Issues** and **Users** together.  
>
> ❌ `Project` is the container to group logically related `Databases`, `Issues` and `Users` together.  


3. Quote for file name, file path, database name, instance name, and etc.

> ✅ The `Employee` project is created. 
>
> ❌ The **Employee** project is created.
>
> ❌ The ”Employee” project is created.


4. Spell the number when it’s less than or equal to ten.

> ✅ one instance
>
> ✅ two databases
>
> ❌ 1 instance
>
> ❌ 2 databases

5. Specify roles and plans at the top of each article if applicable. Use `<hint-block type="info">` component to wrap the sentence up.

> ✅ 
> This feature is only available for 
> - **Workspace Admin** role 
> - **Team** or **Enterprise** plan

6. For a task-based heading, start with a bare infinitive.

> ✅ Create an instance
>
> ❌ Creating an instance

7. For a conceptual or non-task-based heading, use a noun phrase that doesn't start with an -ing verb.

> ✅ Migration to Google Cloud
> 
> ❌ Migrating to Google Cloud

8. Don't introduce a procedure with a partial sentence completed by the numbered steps.

> ✅ To customize the buttons, follow these steps:
> 
> ✅ Customize the buttons:
> 
> ❌ To customize the buttons:

# Readings

- [Docs for Developers](https://docsfordevelopers.com)
- [A Framework for Writing Better Documentation](https://documentation.divio.com/structure)
