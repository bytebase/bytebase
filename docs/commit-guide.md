# Commit Messages Guide

This guide describes how we add human and machine-readable commit messages.

Refer to [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)

## Benefits

* Automatically generating CHANGELOGs and release notes.
* Communicating the nature of changes to teammates, the public, and other stakeholders.
* Making it easier for people to contribute to your projects, by allowing them to explore a more structured commit history.

## Summary

Conventional Commits specification is a lightweight convention on top of commit message. It provides an easy set of rules for creating an explicit commit history, which improves better readability, velocity and automation.

## Commit Message Style

The commit message should be structured as follows:

```bash
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Header

The header is a `mandatory` line that simply describes the purpose of the change.

```bash
<type>[optional scope]: <description>
```

It consists of three parts in itself:

* `Type` - a short prefix that represents the kind of the change
* `Scope` - optional information that represents the context of the change
* `Subject` - represents a concise description of actual change

> 💡 Notice that there also has a colon and space(`:<space>`), which separated the type and description

### Body

The body is optional lines that introduce the motivation behind the change or just describing slightly more detailed information.

### Footer

The footer is optional lines that mention consequences which stems from the change - such as announcing a breaking change, linking closed issues, mentioning contributors and so on.

## Examples

### Commit message with no body

```bash
docs: correct spelling of CHANGELOG
```

### Commit message with scope

```bash
feat(lang): add polish language
```

### Commit message with description and breaking change footer

```bash
feat: allow provided config object to extend other configs

BREAKING CHANGE: `extends` key in config file is now used for extending other config files
```

### Commit message with body and footer

```bash
fix: prevent racing of requests

Introduce a request id and a reference to latest request. Dismiss
incoming responses other than from latest request.

Reviewed-by: Z
```

## Common Types

On top of defining the commit message format, the [Angular commit message conventions](https://github.com/angular/angular/blob/22b96b9/CONTRIBUTING.md#-commit-message-guidelines) specify a list of useful types that cover various sorts of changes.

### feat

Used for add some feature

```bash
feat: add some feature
feat(frontend): add some feature in frontend
```

### fix

Used for fixed some bug

```bash
fix: fixed typo
fix(frontend): fixed some error in frontend scope
```

### docs

Used for write some docs

```bash
docs: add a new docs
```

### refactor

Used for refactor some old code or maybe rewrite、enchance it

```bash
refactor: rewrite some logic
```

### style

Generally used for front-end

```bash
style: update the ui colors
```

### test

Used for add some test case

```bash
test: test some case
```

### perf

Used for some performance improvements.

```bash
perf: reduce the api requests when page loading
```

### chore

Used for changes the build process, bump up version, add some configs etc.

```bash
chore: release v0.0.1
chore: bump up dependencies version
```

Checkout [more examples](https://www.conventionalcommits.org/en/v1.0.0/#examples)

## Tools

* [commitizen](https://github.com/commitizen/cz-cli) - The commitizen command line utility.
* [cz-conventional-changelog](https://github.com/commitizen/cz-conventional-changelog) - A commitizen adapter for the angular preset of [conventional-changelog](https://github.com/conventional-changelog/conventional-changelog)
* [husky](https://github.com/typicode/husky) - Git hooks made easy 🐶 woof!
* [conventional-changelog-cli](https://github.com/conventional-changelog/conventional-changelog/tree/master/packages/conventional-changelog-cli) - Generate changelogs and release notes from a project's commit messages and metadata.
