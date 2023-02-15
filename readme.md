# cleaner

disk protector for programmer (replacement of [purge](https://github.com/ZenLiuCN/purge)).
1. this one use requires go>=1.18 (go.mod requires 1.20)
2. should be able to run any os that go supported (none os spec api used)

## action

1. read `.gitignore` for files should be cleaned.
2. read `.cleanignore` for files should be kept, this has the highest priority.
3. read `{userhome}/.cleaner` as user level config.
4. read `{ExecuteablePath}/.cleaner` as global config.

more detail just `go get github.com/ZenLiuCN/cleaner && go install github.com/ZenLiuCN/cleaner && cleaner`

## license

DO ANYTHING U LIKE BY FORK 