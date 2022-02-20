This folder contains the seed data for release. The release seed data is used when building with "release" tag or running with --demo.

The seed data file name follows the same naming convention as the migration file like {{version_number}}\_\_{{description}}, where {{version_number}}
corresponds to the migration file having the same {{version_number}}. The particular release seed data file is ONLY loaded when the corresponding migration file has been applied, which means it should only be loaded once together with running the migration file.
