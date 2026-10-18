# Desk handbook

This is the how-to for the signed-in company desk at TheMobileProf Technologies (Lagos). One CAC company. Workers propose. Humans stamp. The How? button answers from this file, the FAQ, and the Docs playbooks. It never stamps Accept, never sends mail, never publishes a page, never hires anyone.

## Sign in and what you see

- Open `/admin/`. Owner and BDM land on Board. Everyone else lands on the worker desks ticked on their People card (or a picker if several).
- Login is a People card with a password and at least one desk tick (that tick grants `web_admin`). A Face seat on a project is not a login.
- Project faces and field assistants often have no login. They work from the this-week note on the project card.

## People (owner only)

- Path: `/admin/users`. BDM cannot add people — ask the owner.
- Add: name, phone in `234…` form, email, office seat (`owner`, `bdm`, `assistant`, …), hours/week, then tick **Desks**.
- Hours/week is Delivery pace. Seat the person on a project **Face** tab (cofounder or assistant) to onboard them onto that bet.
- Tick only the workers they will run. Ticking Legal is what stops `/admin/legal` from 403. Same for SEO.
- Rights: `whatsapp_accounts` allowlists that phone for customer WhatsApp `/accounts`, `/ops`, `/legal`, `/seo`. `/today` is Delivery if that number is on People.
- Recruiting is closed. There is no hire stamp. Inbound CVs go to Growth. Do not use HR as a department. Do not treat `/approve` as a hire.

## Board, Projects, Delivery

Do this, in order:

1. Board `/admin/` — look at each card. Open **Proposal waiting** → Stamp → Accept or Reject. Open **Coming up** dates.
2. Projects `/admin/projects` — click a card. Work the tabs: Stamp, then Dates, then Work, then Face. Academy LMS is a bet, not a second company. Ask placement writes a proposal; Accept moves the stamp. Chat never stamps.
3. Delivery `/admin/delivery` — click Done on today’s tasks. To plan a week: Ask plan, then Accept on Job chat. Stamp caps: parked/killed 0, idea 1, testing 3, selling 5, fundable 4. Hours/week on People is pace. Seat on Face so they can receive tasks.
4. Own `/today` check-ins may apply immediately. Accept still belongs to a human for plans the worker proposed.

## How? vs worker Ask

- **How?** (bottom right on every signed-in page) is this guide. It tells you which screen to open and which button to press. It does not file a job.
- Worker **Ask** (on `/admin/desks/{slug}` or a book such as Legal/SEO/Delivery) files a **proposed** job. Then chat a fix if needed. Then **Accept** or Turn away.
- Chat never stamps. The worker JSON never writes `approved`.

## Handler desks

- Path: `/admin/desks/{slug}` after the owner ticked that slug on your card.
- Accounts `/accounts` — Books classifier. n8n posts to Zoho Books and Paystack. Go never computes VAT.
- Growth `/growth` — sales copy and replies. Bare 1:1 chat is Growth.
- Community `/cm` `/community` — Academy Telegram rooms. Mandates at `/admin/community`. Cloud API WhatsApp cannot join ordinary groups.
- CRM `/crm` — pipeline notes. n8n writes Zoho Books contacts, not Zoho CRM.
- Legal `/legal` — NDA and contract proposals. Accept on `/admin/legal`. The worker cannot send. Telegram watch flags labour/CAC/NDPR/CBN without a prefix; it does not save a desk row.
- SEO `/seo` — titles, meta, outlines, audit punch lists. Accept on `/admin/seo`. The worker cannot publish, crawl, or read Search Console.
- Delivery `/today` — see Delivery above. Owner/BDM can open Delivery even without a tick.
- Ops `/ops` — operations room. Accounts via `/ops` still writes Books. `/ops legal` is chat-only; filing is `/legal`.
- Product `/validate` — placement and validation. Journey stamps stay on the project card.

## Docs

- `/admin/docs` — start at How you fit, then the sheet that matches the People card. Worker specimens sit below the player sheets.
- Print from the browser. If the card and the sheet disagree, amend People or the Face tab — do not invent a second company.

## Office menu (owner / BDM)

- Defaults `/admin/settings` — Zoho org and chart-of-accounts ids. Secrets stay in the VM `.env` / n8n Credentials.
- Docs `/admin/docs`.
- n8n `/admin/workflows` — iframe of the editor. Do not treat it as the desk.
- Guide `/admin/guide` — questions people asked How?. Download `/admin/guide/asked.md` and fold durable answers into `faq.md` in Cursor.

## Chat prefixes (not this website)

Do not give each department its own WhatsApp. One ops Telegram, one Academy Telegram, one customer WhatsApp, one `info@`.

| Starts with | Worker |
| --- | --- |
| `/accounts` | accounts (WhatsApp allowlisted) |
| `/ops` | internal-ops (WhatsApp allowlisted) |
| `/legal` | legal (WhatsApp allowlisted) |
| `/seo` | seo (WhatsApp allowlisted) |
| `/today` `/pm` `/idea` | delivery |
| `/growth` | growth |
| `/crm` | crm |
| `/cm` `/community` | community |
| `/validate` | product-dev |
| none, 1:1 | growth |
| none, group | community (Academy Telegram student rooms) |

`/approve` / `/kill` on Telegram only send or drop the pending **info@** draft. They do not Accept a legal, project, delivery, or SEO stamp.

Academy bot refuses ops prefixes. Do not add the ops bot to a class group.

## Hard no

- Do not stamp by chatting.
- Do not publish from the SEO worker.
- Do not send from the Legal worker.
- Do not hire through HR.
- Do not invent a second CAC company for a project face.
- Do not put Zoho/Paystack tokens in git or in the Go worker.
