# Tips for Useful Performance Testing Commands

## Check number of Connections

To see the number of connections to the mattermost server you can run commands like:

```
sudo netstat -an | grep :8065 | wc -l
```

or:

```
ss | grep ESTA | grep 8065
```

## Verify the ulimits are set correctly

You can verify process `myprocess` has the correct amounts by running:

```
cat /proc/`pgrep myprocess`/limits
```
