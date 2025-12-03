# 4. Config location

Date: 2024-12-18

## Status

Accepted

## Context

We might sometimes need to work with multiple configurations looking at multiple environments.
We need a way to be able to easily switch between configs.

Examples:

- CECG staff switching between tenant clients
- If a tenant has set up a set of testing environments with a test repo, they might want to switch

## Options

We can follow linux convention like other tools, or diverge from the XDG standard

### Option 1 - Use XDG Base Directory Specification

[This specification](https://specifications.freedesktop.org/basedir-spec/latest/) defines standard locations for user-specific files and directories,
using environment variables to specify base directories for data, configuration, state, and runtime files.
It also defines an ordered set of directories to search for data and configuration files.
This specification aims to provide a consistent and portable way for applications to store and access user-specific data.

This might result in a file structure like this:

```text
~/.config/myapp/             # Configuration files
    └── config.yaml          # Main configuration file
    └── settings.json        # Additional settings

~/.local/share/myapp/        # Data files
    └── repositories/        # Data related to repositories
        ├── repo1/
        └── repo2/
    └── logs/                # Log files
        ├── 2023-12-01.log
        └── 2023-12-02.log

~/.cache/myapp/              # Cache files
    └── temp/                # Temporary data
    └── precomputed/         # Cached computations
```

The files for myapp are put in conventional places, but not together.

### Option 2 - Copy the style of kubernetes, keep everything in one folder

This means everything is in one place, easy to locate and manage.
This would result in a folder structure like this:

```text
~/.myapp/                    # All files in one place
    ├── config.yaml          # Main configuration file
    ├── settings.json        # Additional settings
    ├── repositories/        # Data related to repositories
    │   ├── repo1/
    │   └── repo2/
    ├── logs/                # Log files
    │   ├── 2023-12-01.log
    │   └── 2023-12-02.log
    ├── cache/               # Cached computations
    │   ├── temp/
    │   └── precomputed/
    └── temp/                # Temporary files
```

The key benefit of this being consistency with kubernetes, which likely 100% of users will be familiar with.
The downsides are minimal to this approach, so long as we provide support for override paths.

## Decision

We should stick with option 2. This is the setup that will feel most familiar to developers and users of the tool.
Any downsides are minimal, and we can mitigate them through proper tooling.
