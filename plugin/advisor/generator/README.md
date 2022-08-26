This is a SQL review rule generator. It's used for implementing a specific SQL review rule.

## How To Use

1. Add the `Advisor Type` in `/plugin/advisor/advisor.go`:
   ```go
   const (
    // ...

        // MySQLColumnDisallowChangingType is an advisor type for MySQL disallow changing column type.
	    MySQLColumnDisallowChangingType Type = "bb.plugin.advisor.mysql.column.disallow-changing-type"
   )
   ```
   You need write both code and the comment.
2. build the generator.
   ```shell
   go build
   ```
   in `/plugin/advisor/generator`
3. run generator to generate the framework code.
   ```shell
   ./generator --flag {AdvisorType}
   ```
   in `/plugin/advisor/generator`
   e.g.
   ```shell
   ./generator --flag MySQLColumnDisallowChangingType
   ```
4. Implement the rule-specific logic in the generated files.
