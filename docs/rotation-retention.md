# Rotation and retention

**Issue**: `{id}.log` grows forever and every read scans all of it. Old data needs to be dropped without breaking offset reads or ID continuity.

**Initial Approach**: A single ever-growing active file per service, with no cleanup.

**Final Approach**:
- Naming: the active file stays `foo.log`. Rotated segments are `foo.log.YYYYMMDD`, or `foo.log.YYYYMMDD.N` when a day needs several. The date is the day the data was written, not the day of rotation, so the filename alone says how old the data is.
- Check runs on every append, before the write. It rotates when the calendar day differs from the segment's day, or when the active file's size is `>= GOZZLE_WAL_MAX_BYTES`.
- Rotation closes the active descriptor, rename to the next free dated name, reopen a fresh `foo.log`, reset size and day, then run retention.


**Trade-offs**:
- The size check is before the write, so a segment can exceed the limit by one entry.
- Retention only runs on rotation and restart, so an idle service's expired segments linger until one of those happens.

**Plan for Improvements**: Run retention on a timer. Hold the lock (or a read lock) while listing segments. Of maybe run rotention/rotation asynchronously with a wait flag so that during that time, writes are queued. 
