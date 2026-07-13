# Bloom Filter

## What Is It?

A **Bloom Filter** is a space-efficient probabilistic data structure that
answers set-membership queries with the guarantee:

- **False negatives**: IMPOSSIBLE — if `MightContain` returns `false`, the key
  is **definitely absent**.
- **False positives**: Possible — `MightContain` may return `true` for a key
  that was never inserted.

### Structure

A bit array of `m` bits + `k` independent hash functions:

```
bit array (m=16, k=3), after inserting "foo":

Index:  0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15
Bits:   0  0  1  0  1  0  0  1  0  0  0  0  0  0  0  0
             ↑        ↑        ↑
          h1("foo") h2("foo")  h3("foo")
```

## Complexity

| Operation | Time | Space |
|---|---|---|
| Add | O(k) | O(m) total |
| MightContain | O(k) | O(1) extra |

## Significance in Databases

Used widely to gate disk/network I/O operations for non-existent keys:
- **Apache Cassandra** — per-SSTable Bloom Filters eliminate 99% of disk reads for absent keys.
- **RocksDB / LevelDB** — per-SST Bloom Filters gate read operations at each level.
- **Google Bigtable** — original paper mentions Bloom Filters for SSTable lookups.

## How It Is Used Here

In DocDB, the Bloom Filter (`internal/ds/bloom.go`) is integrated into each
**SSTable** run inside the LSM Tree. When looking up a document key in Level 0 or Level 1,
DocDB checks the Bloom Filter of each SSTable. If it returns false, the binary search
over that file is skipped entirely, saving valuable CPU cycles.
