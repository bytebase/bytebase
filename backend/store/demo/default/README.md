Each table corresponds to a file, the file name has the format of {{number}}##{{table_name}}. The
`number` serves several purposes:

1. Specifies the order of tables to be applied.
1. Indicates the starting id sequence for that table. Thus each table has different id range and
can spot errors if we use the id from a wrong table in the foreign key column.