# Log-Structured Merge Tree (LSM Tree)

## What Is It?

A **Log-Structured Merge Tree (LSM Tree)** is a write-optimized data structure
designed for high-throughput write workloads. It buffers writes in a sorted
in-memory structure called a **memtable** and periodically flushes them to disk
as immutable sorted files called **SSTables (Sorted String Tables)**.

```
                  Write Operations (Put / Delete)
                                │
                                ▼
                     ┌──────────────────────┐
                     │  Memtable (InMemory) │
                     └──────────┬───────────┘
                                │ flush (when threshold reached)
                                ▼
                     ┌──────────────────────┐
                     │   L0 SSTables (Disk) │
                     └──────────┬───────────┘
                                │ compaction (merge sort)
                                ▼
                     ┌──────────────────────┐
                     │   L1 SSTables (Disk) │
                     └──────────────────────┘
```

### Key Properties
- **Append-only writes** — write operations are sequential, avoiding random disk seeks.
- **Deletes are tombstoned** — deleting a key writes a tombstone marker instead of deleting on disk immediately.
- **Compaction** — background processes merge-sort overlapping files, discarding duplicate keys and tombstones.

## Complexity

| Operation | Average | Worst Case |
|---|---|---|
| Write (Put) | O(1) (with skip list / memtable insert) | O(1) |
| Read (Get) | O(log n) (depending on level count) | O(L * log n) (where L = level count) |
| Range Scan | O(L * log n + k) | O(L * log n + k) |

## Significance in Databases

The LSM Tree is the standard storage engine structure for write-heavy NoSQL databases, including:
- **RocksDB / LevelDB** — embedded key-value engines used in CockroachDB and TiDB.
- **Apache Cassandra** — wide-column store using LSM Tree structures.
- **InfluxDB** — time-series database.
- **MongoDB** — uses WiredTiger LSM engine for write-heavy collections.

## Trade-offs

| Pro | Con |
|---|---|
| Extremely fast write performance (sequential I/O) | High write amplification (compaction rewrites data) |
| Higher storage efficiency (no fragmentation) | Reads are slower (must check multiple levels) |
| Lock-free friendly memtable implementation | Compaction spikes can impact latency consistency |

## How It Is Used Here

In DocDB, the LSM Tree (`internal/ds/lsmtree.go`) serves as the core persistence
model. When documents are inserted, updated, or deleted, they are stored in the
memtable. The memtable flushes to L0 when it reaches the threshold capacity. L0 tables
undergo compaction into Level 1 when there are too many files.
Each SSTable incorporates a **Bloom Filter** to avoid binary searching files that
do not contain the queried key.
