# casper

[![Docker Cloud Automated build](https://img.shields.io/docker/cloud/automated/backwardspy/casper?style=for-the-badge)](https://hub.docker.com/r/backwardspy/casper) [![Docker Cloud Build Status](https://img.shields.io/docker/cloud/build/backwardspy/casper?style=for-the-badge)](https://hub.docker.com/r/backwardspy/casper/builds)

he lives again by popular demand.

## usage

```bash
$ casper -token $BOT_TOKEN -guild $GUILD_ID -dbPath $DATABASE_PATH
```

omit `-guild` if you want the commands to be registered globally.

omit `-dbPath` to use `./casper.db`.

## commands

### meatball day

`/meatball [USER]` looks up a user's meatball day in the meatball day database.

`/meatball-save MONTH-DAY` save your meatball day into the meatball day database.

`/meatball-forget` remove your meatball day from the database.

`/meatball-role ROLE` set the role to assign on meatball day. **\[admin only\]**

`/meatball-chan CHANNEL` set the channel to use for announcements. **\[admin only\]**
