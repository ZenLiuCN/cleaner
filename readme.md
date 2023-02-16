# cleaner

disk protector for programmer (replacement of [purge](https://github.com/ZenLiuCN/purge)).
1. compile requires go>=1.18 (go.mod requires 1.20)
2. should be able to run any os that go supported (none os spec api used)

## action

1. read `.gitignore` for files should be cleaned.
2. read `.cleanignore` for files should be kept, this has the higher priority.
3. read `{userhome}/.cleanignore` as user level config.
4. read `{ExecuteablePath}/.cleanignore` as global config.

**note**: global and user level config can only process final files or folders,can not match with a relative path.
common use case should to define some common trash or important files those aren't pushed to git.
more detail just `go get github.com/ZenLiuCN/cleaner && go install github.com/ZenLiuCN/cleaner && cleaner`
## sample global .cleanignore
```ignore
# keeps git and ide config

!.idea/
!.git/
!.vscode/
!*.iml
!*.gradle
!pom.xml

# archive

!*.7z
!*.rar
!*.zip
!*.tar*

# backups

!.bk/
!.backups/

# build tools caches should clean
target/
.bin/
cmake-build-debug*/
node_modules/
# common src skip deep scan to speed up process
!src/


```
## license

DO ANYTHING U LIKE BY FORK 