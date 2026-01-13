# gitsaver

A simple tool to back up your Git repositories.

## Providers

Currently, gitsaver supports backing up repositories from GitHub.

## Configuration

The application can be configured using environment variables.

| Variable                           | Description                                                                          | Default                          |
| ---------------------------------- | ------------------------------------------------------------------------------------ | -------------------------------- |
| `DESTINATION_PATH`                 | The path where the backups will be stored.                                           | `./output` and "/ouput in Docker |
| `PORT`                             | The port on which the application will run.                                          | `8080`                           |
| `GITHUB_BACKUP_METHOD`             | The backup method to use. Can be `git` or `tarball`.                                 | `tarball`                        |
| `GITHUB_RUN_ON_STARTUP`            | Whether to run the backup on startup.                                                | `false`                          |
| `GITHUB_CRON`                      | The cron expression for scheduling backups.                                          | /                                |
| `GITHUB_TOKEN`                     | The GitHub token to use for authentication.                                          | /                                |
| `GITHUB_USERNAME`                  | The GitHub username.                                                                 | /                                |
| `GITHUB_INCLUDE_OTHER_USERS_REPOS` | Whether to include repositories from other users.                                    | `false`                          |
| `GITHUB_INCLUDE_FORKED_REPOS`      | Whether to include forked repositories.                                              | `false`                          |
| `GITHUB_INCLUDE_ARCHIVED_REPOS`    | Whether to include archived repositories.                                            | `false`                          |
| `GITHUB_EXTRACT_TARBALLS`          | Whether to extract the tarballs. Only used when `GITHUB_BACKUP_METHOD` is `tarball`. | `false`                          |


## Docker

```yaml
services:
  gitsaver: 
    image: ghcr.io/zareix/gitsaver:latest
    environment:
      - DESTINATION_PATH=/output
      - PORT=8080
      - GITHUB_BACKUP_METHOD=tarball
      - GITHUB_RUN_ON_STARTUP=true
      - GITHUB_CRON=0 0 * * *
      - GITHUB_TOKEN=your_github_token
      - GITHUB_USERNAME=your_github_username
      - GITHUB_INCLUDE_OTHER_USERS_REPOS=false
      - GITHUB_INCLUDE_FORKED_REPOS=false
      - GITHUB_INCLUDE_ARCHIVED_REPOS=false
      - GITHUB_EXTRACT_TARBALLS=false
    volumes:
      - ./output:/output
    ports:
      - 8080:8080
```
