# Point-in-time Recovery

*Note: implementation details in this design doc focus exclusively on MySQL, while the general design should also apply to PostgreSQL.*

# Terminology
A *physical backup* consists of raw copies of the directories and files that store database contents. This type of backup is suitable for large, important databases that need to be quickly recovered when problems occur. 

A *logical backup* is a logical snapshot of a database, represented by a full logical dump of SQL statements. [This reference](https://dev.mysql.com/doc/refman/8.0/en/backup-types.html) highlights the difference between logical backup and physical backup.

*Incremental backup* is the changes to the database since the last full backup, represented as the binlog in MySQL. Incremental backup can also be called data changes. Incremental backup applies to logical backups.

*Point-in-time Recovery* is the process of making a database's state current up to a given time by restoring the database from a logical backup and applying incremental backup data changes up to the given time.

A *database* is a collection of tables represented as is in MySQL. A *database instance* (instance for short) is the server process that runs the database management system. In MySQL, it’s similar to a `mysqld` process.


# Overview

Currently, Bytebase users can set up a backup policy, take full logical backups automatically following the backup policy schedule, and restore the backup to a new database. There are two problems we would like to address in this design.

1. Point-in-time Recovery.
2. In-place recovery.

We are introducing Point-in-time Recovery (PITR). Consider the scenario that the user made a mistake and dropped a table at 8:00 am, only to find that the last full backup was produced at 2:00 am. If they restore to the previous backup snapshot, new data changes would be lost in the latest 6 hours. With PITR, we can restore the database state to any point between the checkpoints at the granularity of a single database transaction or a timestamp.

We are also introducing In-place Recovery. Without it, users have to change the data source name (DSN) and the database connection string in the software, which is a pain point because it requires

1. Coordination between developers, DBAs, and even network engineers,
2. Source code modification and software redeployment.

We target database-level recovery with logical backup because the database is the minimal isolation unit within a database instance (an instance of a database server), a natural, logical unit mapping to the business application unit. Logical mistakes usually happen to a single database. Users would like to recover a single database without impacting other databases.

## Requirements

- MySQL 8.0 and 5.7.
- The database should use the InnoDB storage engine.
- Enable binlog and choose row-based binlog format.
- Databases for Point-in-time Recovery are not involved in [XA Transactions](https://dev.mysql.com/doc/refman/8.0/en/xa.html).

Note that when statement binlog format is used, the `mysqlbinlog` tool could misbehave with regard to the `--database` option as described [in the docs](https://dev.mysql.com/doc/refman/8.0/en/mysqlbinlog.html#option_mysqlbinlog_database). In short, using statement logging has a chance to miss some binary log events in the specified database.


# Critical User Journeys

This section describes the common critical user journeys about why and how the users may use the Point-in-time Recovery feature to recover their data.

## OMG Moments

Occasionally, catastrophic events can happen to the databases and result in service downtime. Here are some cases where the users would like to recover the database to a specific point in time.

- **Drop Database**: the database is dropped accidentally by mistake.
- **Schema Migration Failure**: A column is dropped, or its type is updated incorrectly. The table needs to be rolled back to the state before the migration. In the long term, we could polish the feature to provide the ability to revert the migration with minimal cost. Still, currently, we could leverage PITR to do the lift, with the limitation of losing incoming data after the migration.
- **Buggy Application**: Bugs happen, and they may corrupt your data. The chances are that the user rolls out a new release of their service, only to find a bug in the application code that deletes essential data. It is best to roll back the service and recover the database to the state just before the release.

## Database Recovery

When OMG moments happen, the user would like to recover the database to the latest correct state as soon as possible. This is where the Point-in-time Recovery feature kicks in.

As long as the instance fulfills all the [requirements](#requirements) and enables a backup policy in the Bytebase environment that contains this instance, the user is lucky enough to have a PITR option. The user could then choose those databases they want to recover, and select the point of time, then Bytebase will recover the databases to precisely the state of that point of time.

What’s more convenient is that if the user used Bytebase to do the wrong schema migration, then Bytebase will record the exact time before the schema migration and provide a one-click Point-in-time Recovery experience.


# Design

## Point-in-time Recovery Process

### Full Backup

Full backup follows what `mysqldump` does with `--source-data` flag. `SHOW MASTER STATUS` gives the coordinate (MySQL binlog file name and position) of a backup. This provides us with the starting position to apply binary logs in the PITR recovery step. Because the `SHOW MASTER STATUS` statement is not transactional, we have to put [`FLUSH TABLES tbl_names... WITH READ LOCK`](https://dev.mysql.com/doc/refman/8.0/en/flush.html#flush-tables-with-read-lock-with-list) to block writes to figure out the precise binary log coordinate in the transaction of taking database backup. As a side-effect, writes will be blocked for a short period between steps 1 and 4. By the way, we have to make sure the tables are unlocked regardless of any failure.

1. `FLUSH TABLES tbl_names... WITH READ LOCK`.
2. `SHOW MASTER STATUS`. Bytebase will record the binary log coordinates and timestamp.
3. Start a transaction. This [will implicitly unlock tables](https://dev.mysql.com/doc/refman/8.0/en/lock-tables.html) that were locked in the same session.
4. Dump all the schema and data for a database.
5. Commit or abort the transaction.

Step 1 is different steps from a typical automatic database backup that could affect database performance. But since it only affects a single database for a short period of time, it is acceptable in most cases.

Step 4 should write the binlog file name and position (a.k.a binlog coordinates) obtained in step 2 in the same logical backup file.

An example setup of a backup policy is to back up the database every day at 2 am when the traffic is low and continuously store the data changes across the whole day. If the backup interval is one day, then there should be a series of backup start points in the view of the timeline.

The logical backup and data changes, i.e., binlog, should be kept in safe and durable storage, e.g., AWS S3 or NFS. Note that users should not manually manage the storage, introducing inconsistency between the metadata in Bytebase and storage contents.

Currently, we keep the logical backup and binlog together in the local disk where Bytebase resides. We may integrate with cloud storage in the long term.

The partial backup dump should be discarded if Bytebase encounters an unrecoverable error during the full backup process. The system should report a failed backup, and the user could choose to start another backup task manually. The partially dumped logical backup file will be automatically deleted by Bytebase.

To check the integrity of the logical dump, Bytebase will write file size along with the table data into the backup file. When using a logical backup to recover, we should first validate the row count.

## Incremental Backup

Once the backup is enabled for a database, Bytebase will periodically download binlog from the MySQL server via `mysqlbinlog` with parameter [`--read-from-remote-server`](https://dev.mysql.com/doc/refman/8.0/en/mysqlbinlog.html#option_mysqlbinlog_read-from-remote-server), which reads data via the [binlog network stream](https://dev.mysql.com/doc/internals/en/binlog-network-stream.html). The binlog downloading process is described below.

1. Check the local existing binlog files.
2. Obtain a listing of the binary log files on the MySQL server with SHOW BINARY LOGS.
3. Download the previous except the current binlog files on the MySQL server that does not exist locally.

The process above should be executed every minute. When a download task fails, Bytebase should delete the corresponding binlog file. The downloading process must be idempotent.

To accomplish the tasks described above, Bytebase will embed the newest version of `mysqlbinlog` binary, which is also used in the recovery process.

When the backup is enabled for a specific database, Bytebase will check where the PITR requirements are met and automatically upgrade to PITR and download the binlog files.

An example command is like this, and the `--ssl` parameters should be used [following the doc](https://dev.mysql.com/doc/refman/5.7/en/mysqlbinlog.html#option_mysqlbinlog_ssl) if required:

`mysqlbinlog mysql-bin.000001 --ssl=0 --read-from-remote-server --host=x --user=x --password=x --raw --result-file=/bytebase/binlog/`

## Find the point-in-time T

The user should be able to navigate the database history to search for SQL queries to determine the exact point in time to recover. As is described in the [OMG Moments](#omg-moments), when the user meets a schema migration failure, Bytebase should locate the binlog/wal position just before the corresponding migration happens.

Bytebase will support a binlog search api accepting a timestamp and returns several numbers of events around it, containing the data changes and binlog positions. The user should choose a proper binlog position from the UI so that Bytebase can recover the database to the corresponding state.

MySQL binlog contains information about the ALTER TABLE statement. When a schema migration happens in Bytebase, we should search the binlog and find the position of the schema migration. If the user wants to revert the schema migration, this position should be the target position of a later PITR. The user can revert it with a simple button click.

## Recovery

When the point-in-time t1 is determined, Bytebase will take a series of actions to accomplish the task. The recovery plan is as follows.

1. Locate the logical backup just before t1, which is called bk and represents a database snapshot at t0.
2. Create a temporary database called pitr, with the naming convention described below.
3. Restore logical backup bk to the pitr database, after which the pitr database should be a snapshot at t0 of the original database.
4. Apply data changes from t0 to t1.
5. Validate the pitr database to make sure everything is as expected.
6. Abort or execute the cutover.
    1. Abort the recovery will delete the pitr database.
    2. Execute the cutover: if the original database exists, rename the original database to old, and rename the pitr database to the original; if not exists, we just create the pitr table using the user-provided name.

The timeline involved above could be roughly represented like this:
(bk (t0), data changes, t1, data changes)

The pitr database naming convention in step 2 is appending _pitr_${UNIX_TIMESTAMP} to the end of the original database name. We must check that the pitr database name does not contain more than 64 characters, which is [the limit of MySQL](https://dev.mysql.com/doc/refman/8.0/en/identifier-length.html).

Step 6b uses a special MySQL syntax [`RENAME TABLE current_db.tbl_name TO other_db.tbl_name`](https://dev.mysql.com/doc/refman/8.0/en/rename-table.html), which effectively moves tables between databases. At last, Bytebase will delete the pitr database, which will be empty. The original database, if it exists, will be deleted if the user manually approves the issue Bytebase automatically generates.

After the recovery process described above, Bytebase will ask for the user’s manual approval to delete the old database.

To restore binlog from the original to the pitr table as is described in step 4, Bytebase will use the [`--rewrite-db`](https://dev.mysql.com/doc/refman/8.0/en/mysqlbinlog.html#option_mysqlbinlog_rewrite-db) option. An example is like `mysqlbinlog --rewrite-db='dbtarget->dbtarget_pitr' --database=dbtarget_pitr binlog.00001`. The `--database` option filters changed database names, so should not use the original database name.

The database session used to accomplish this recovery task should disable binlog/wal.

After the recovery process, the user has the original database named old to investigate for problems or just delete to reclaim storage space. The original database is recovered to the point in time.

Before the cutover stage, Bytebase will do several checks to prevent potential damage:
- Check that no connection exists on the original database

## Crash
If the recovery process crashes before completion, we should mark the recovery as failed. The user should start a new recovery process.


# API and Schema

Bytebase should make some changes to the API and schema design to implement the PITR feature.

## Backup PITR Information

When taking a backup of a database, now we also need to store the current position in the data change stream when the logical backup is dumped. In MySQL, this position refers to binary log file name and position. In PostgreSQL, it’s the WAL lsn. To use a single model, we store a JSON encoded string that could be encoded and parsed, respectively.

To provide a search option with the timestamp and binlog position, we will encode both of them, where the timestamp is the time at which the statement began executing on the MySQL server and decoded from the [binlog event header](https://dev.mysql.com/doc/internals/en/event-header-fields.html). The timestamp is represented as the UNIX timestamp, i.e., the number of seconds since 1970 (UTC).

An example of the PITR information is like this:

```json
{
    "binlog_info": {
        "binlog_name": "binlog.000001",
        "binlog_position": 1234,
        "created_ts": 1650957790
    }
}
```

We will add a field in api.Backup:

```go
type Backup struct {
	...
	// Payload contains arbitrary string message with following cases
	// 1. contains the starting position of incremental backup when this backup snapshot is taken.
	// This field is not returned to the frontend.
	Payload string
}
```

We will add a column in table backup:

```sql
CREATE TABLE backup (
    ...
    payload TEXT NOT NULL
);
```

When the user performs a PITR recovery, Bytebase will check the api.Backup and try to find the PITR information. If the PITR information is not empty, Bytebase will perform a binlog replay after recovering the logical backup using the [recovery process described above](#recovery).


# Data Retention Policy

As time goes by, the logical backup and incremental backup will take up infinite storage space, and a data retention policy should be defined to solve this problem.

Bytebase will currently implement a default data retention policy of 7 days. We may later make this configurable as an improvement.


# Monitoring

Steps 3 and 4 could take some time, and Bytebase should export the progress via api.Task, which could be obtained from an api.Issue.


# Third-Party Integration

Bytebase should provide event webhooks for user system integration. For example, we could send the logical backup success/failure events to a user-defined URL.


# Test Plan

To test the Point-in-time Recovery is working as expected, we will write several unit test cases following the [Critical User Journeys](#critical-user-journeys) described above. The test cases should cover major path and corner cases as possible.
