# Log ID assignment

**Issue**: Finding the ID of the next log entry

**Initial Approach**: on startup, scan the WAL and set `nextID = line count + 1`. But it does not scale with increasing number of log entries with `O(n)` runtime, and also when the log files are rotated. 


**Final Approach**: Store `next_id` in a small JSON file, separate from the log. Startup becomes `O(1)` and independent of log contents.

```json
{"services":{"foo":{"next_id":4},"bar":{"next_id":2}}}
```

**Trade-offs**: The whole registry is rewritten on every append, so cost grows with the number of services, and it runs under the store-wide mutex. Two parallel requests from two services (`foo` and `bar`) arrive, the JSON is completely rewritten each time:

```json
{"services":{"foo":{"next_id":5},"bar":{"next_id":2}}}
```

and then,

```json
{"services":{"foo":{"next_id":5},"bar":{"next_id":3}}}
```

**Plan for Improvements**: Maybe incorporate a local SQLite DB or use periodic flush. Haven't decided yet. 