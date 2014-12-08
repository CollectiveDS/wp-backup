wp-backup
=========

A Golang project to backup a static copy of a Wordpress site either to a local directory (for a local backup) or an AWS S3 bucket (for backup and a failover domain).

## Flags

-h: Display help document.

-dest="": Destination local directory or S3 bucket.

-domains="": The domain of the Wordpress site to archive.

-max="1000": The maximum amount of pages to crawl on the site.

-debug="": If set, prints debug statements.