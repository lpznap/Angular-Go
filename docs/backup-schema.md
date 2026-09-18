# Backup schema, version 1

The root object is `{ "schemaVersion": 1, "notes": [...], "attachments": [...] }`.
Note properties match the OpenAPI `Note` schema. IDs are 32 lowercase hexadecimal
characters. Work dates are calendar dates, without timezone conversion. The server
sanitizes descriptions and validates all imported notes again. Unknown JSON fields,
trailing documents, duplicate IDs, invalid dates, and unsupported schema versions fail
validation. Maximum: 500 notes and 500 attachments.

JSON exports intentionally contain an empty attachments array. Use ZIP for binary
attachments. ZIP contains `backup.json`, `report.pdf`, and selected files under
`attachments/<attachment-id>`. Each attachment record contains `id`, `noteId`, `name`,
`mime`, `size`, and `archivePath`. Internal filesystem names and user IDs are never
exported. Every reference must resolve to a file with matching size and type.

The importer reads archives in memory without extracting archive paths. Absolute,
traversal, backslash, drive-letter, duplicate, and nonregular entries are rejected.
Limits: 100 MiB upload/decompressed total, 50 MiB per ZIP entry, 1,001 archive entries,
and the configured attachment size limit. ZIP headers and actual reads are bounded.

Preview validates all contents and returns note/attachment counts and owned duplicate
IDs. The import endpoint repeats validation; it never trusts a previous preview.
Skip leaves existing notes and files untouched. Replace updates existing notes,
increments their versions, and replaces their entire attachment set. **Replacing
from a notes-only JSON removes the duplicate note's existing attachments.** The UI
explicitly names this behavior before confirmation. Foreign ID collisions receive
new IDs and can never overwrite another user's note.

Imports run in one database transaction. A failed batch rolls back all database
changes; successfully staged new files are removed on a failed attempt. Old files
are queued for deletion in the same database transaction as metadata changes.
The background cleanup worker retries deletion failures. New storage IDs are used
for restored attachments. Imported version numbers are never trusted.

See `sample-backup.json` for English and Thai sample data. There are no seeded
passwords or shared demo accounts.
