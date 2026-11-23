# Live asks are not this file

Questions people type into **Help?** are stored in Postgres (`guide_asks`) on the VM. Git cannot see them.

When you come back to Cursor with real questions:

1. Signed in as owner or BDM, open Docs → Asked (`/admin/guide`) or download `/admin/guide/asked.md`.
2. Fold unanswered or weakly answered items into `faq.md` (a `##` question plus a short how-to).
3. If the playbook was wrong, fix `playbook/*.html` and this handbook too — the bot embeds those files.

Do not paste production names into git unless the founder wants them there. Durable how-to belongs in `faq.md`.

Staff Accept and Turn away sit on Docs → Episodes (`/admin/episodes/episodes.md`). Fold a repeating Turn-away reason the same way. Academy Telegram and customer WhatsApp are not in that file.
