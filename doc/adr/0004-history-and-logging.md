# app create - add new application to existing repository

Date: 2024-12-13

## Status

Accepted

## Context

We're analysis our logging requirements. We want to have the history of commands that were run on corectl, 
to facilitate diagnostic of client issues. 
That way, if there is an unexpected error, the user can send us the log files and we
can see the commands the user ran, which arguments were used and what was the exact error.



### Considerations

Considered a specific channel just for this history/audit but it would duplicate things a lot. We decided it would make more sense to have
logging write to both console and a file, where the file would have `info` level, and console `warn` level. This will ensure that only relevant
information ends up in the log file and library debug logs will not go there for the case we have debug level, which is why we decided to have info

### Requirements

There are a few requirements:
* We should be able to log to file as well as to console with different logging levels
* Log level to the console level should be configurable
* log file should be max sized, with retention period and max backups.

### Options

For the requirements our current logging provider does not satisfy them. One that does and is highly used in the community 
is [zap](https://github.com/uber-go/zap), which seems like the way to go.