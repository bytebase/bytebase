This is a SQL review rule generator. It's useful to implement a SQL review rule.

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
2. run
   ```shell
   go build
   ```
   in `/plugin/advisor/generator`
3. run
   ```shell
   ./generator --flag {AdvisorType}
   ```
   in `/plugin/advisor/generator`
   e.g.
   ```shell
   ./generator --flag MySQLColumnDisallowChangingType
   ```
4. write the core code in generated files.
