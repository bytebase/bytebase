# Version Management

Bytebase adopts [Semantic Versioning](https://semver.org/) using the MAJOR.MINOR.PATCH format.

Bytebase ties the version number with the underlying database schema progression, because:

1. Schema change is a good approximate to the functional change. Large schema changes often indicate large functional changes.
1. Schema change determines the customer involvement when upgrading to the new version.

For Bytebase, MINOR and PATCH version upgrade should be transparent to the customer, while MAJOR version upgrade _might_ require manual effort from the customer.

## When MAJOR version is changed

1. Significant product upgrade.
1. Require customer to use a separate method to upgrade the deployed Bytebase version.

MAJOR version change usually happens at most once or twice a year. And if we do, Bytebase will always accomplish the 1st point (delivering great value to the customers) while try to avoid the 2nd point (disrupting the customer).

## When MINOR version is changed

We change MINOR version if the new version upgrades the underlying database schema. While the upgrade does not require customer involvement.

MINOR version change usually happens about once every month.

## When PATCH version is changed

We change PATCH version if the new version does not include underlying database schema changes.

PATCH version change usually happens bi-weekly following our release schedule.
