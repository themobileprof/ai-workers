# Desk handbook

This is the how-to for the signed-in company desk at TheMobileProf Technologies (Lagos). One CAC company. Workers propose. You click **Accept** on the **Decide** tab (that yes used to be called a stamp). The **Help?** button answers from this file, the FAQ, and the Docs playbooks. It never Accepts, never sends mail, never publishes a page, never hires anyone.

## Sign in and what you see

- Open `/admin/`. Owner and BDM land on Board. Everyone else lands on the worker desks ticked on their People card (or a picker if several).
- Login is a People card with a password and at least one desk tick (that tick grants `web_admin`). A Face seat on a project is not a login.
- Project faces and field assistants often have no login. They work from this week’s mission on the project Card.

## People (owner only)

- Path: `/admin/users`. BDM cannot add people — ask the owner.
- Add: name, phone in `234…` form, Telegram id from ops bot `/start`, email, office seat (`owner`, `bdm`, `assistant`, …), hours/week, then tick **Desks**.
- Hours/week is Delivery pace. Seat the person on a project **Face** tab (cofounder or assistant) to onboard them onto that bet.
- Tick only the workers they will run. Ticking Legal is what stops `/admin/legal` from 403. Same for SEO. Those ticks also gate the ops Telegram bot.
- Customer WhatsApp is not allowlisted. Anyone can message it. Books, legal, SEO, and `/today` stay on ops Telegram.
- Recruiting is closed. There is no hire Accept. Inbound CVs go to Growth. Do not use HR as a department. Do not treat `/approve` as a hire.

## Board, Projects, Delivery

Do this, in order:

1. Board `/admin/` — look at each card. Open **Proposal waiting** → **Decide** → Accept or Reject. Open **Coming up** dates.
2. Projects `/admin/projects` — click a card. **Run** is the wizard: same base kit for every bet (Card, Face, Decide, Dates, Work), then this gate’s mission, then Legal / Files / SEO as needed. Help? can fill a step. Ask placement writes a proposal; Accept on Decide changes the stage. Chat never Accepts. Academy LMS is a bet, not a second company.
3. Delivery `/admin/delivery` — click Done on today’s tasks. Ask plan writes after a delay unless you Turn away. Extra tasks sit on Backlog. Fathom notes use the same path. How many open: parked/killed 0, idea 1, testing 3, selling 5, fundable 4. Hours/week on People is pace.
4. Own `/today` check-ins may apply immediately. `/stop` on ops Telegram drops a delayed info@ draft and a delayed Delivery plan. Calendar titles should name the bet slug so notes pin the right card. The 08:30 Telegram ping is one card (tasks, Coming up, CRM due for owner/BDM).

## Help? vs worker Ask

- **Help?** (bottom right on every signed-in page) is this guide. It tells you which screen to open, **or** fills the same form as the page (dates, tasks, People, Ask…). The LLM matches what you meant to the form catalog; Submit posts that form. It never Accepts.
- Worker **Ask** (on `/admin/desks/{slug}` or a book such as Legal/SEO/Delivery) files a **proposed** job. You can type that form on the page or walk it in Help?. Then chat a fix if needed. Then **Accept** or Turn away.
- Chat never Accepts. The worker JSON never writes `approved`.

## Handler desks

- Path: `/admin/desks/{slug}` after the owner ticked that slug on your card.
- Accounts `/accounts` — Books classifier. n8n posts to Zoho Books and Paystack. Go never computes VAT.
- Growth `/growth` — sales copy and replies when you ask. Bare 1:1 chat is Growth. Inbound info@ is CRM.
- Community `/cm` `/community` — Academy Telegram rooms. Mandates at `/admin/community`. Cloud API WhatsApp cannot join ordinary groups.
- CRM `/crm` — Office → CRM is the conversion book. WhatsApp, info@, calls, and `/crm` file a timeline on the person. `/crm follow up Apex` proposes the next step from that story. The 08:30 Telegram card lists due dates to owner/BDM. Won stays for a later upsell. It does not send.
- Legal `/legal` — NDA and contract proposals. Accept on `/admin/legal`. The worker cannot send. Telegram watch flags labour/CAC/NDPR/CBN after another worker answers. Caution and stop also file a Legal job.
- SEO `/seo` — fetches the live URL. Walk is facts, FAQ, schema, onsite pack, then offsite. Ask on `/admin/seo` issues a markdown URL for Cursor. Offsite work is a Delivery week plan that writes after the delay unless you Turn away. Search Console ids on Defaults are your property only.
- Files `/file` — data room. Drop a PDF on ops Telegram. A catalog match files itself. Odd paths stay on `/admin/files` for Accept. n8n writes Google Drive. The worker cannot upload or invent a folder.
- Delivery `/today` — see Delivery above. Owner/BDM can open Delivery even without a tick.
- Ops `/ops` — operations room. Accounts via `/ops` still writes Books. `/ops legal` is chat-only; filing is `/legal`.
- Product `/validate` — placement and validation. Accept on the project **Decide** tab writes the stage.

## Docs

- `/admin/docs` — on the rail for everyone signed in. Start at How you fit, then the sheet that matches the People card. Worker specimens sit below the player sheets.
- Print from the browser. If the card and the sheet disagree, amend People or the Face tab — do not invent a second company.
- Asked `/admin/guide` (owner / BDM, on the Docs rail) — questions people asked Help?. Download `/admin/guide/asked.md` and fold durable answers into `faq.md` in Cursor.

## Office menu (owner / BDM)

Office is the company books, not the library and not n8n.

- CRM `/admin/crm` — conversion book. WhatsApp, info@, calls, and `/crm` file a timeline on the person. `/crm follow up Apex` proposes the next step. The 08:30 Telegram card lists due dates. Won stays for a later upsell. It does not send.
- Files `/admin/files` — odd data-room paths. A catalog match files itself. Accept writes Drive via n8n.
- Grants `/admin/grants` — deadline and next action. `/ops` can file a row. It does not submit an application.
- Defaults `/admin/settings` — one Save per vendor tab (Company, Zoho, Paystack, Drive). Saving one tab does not wipe the others. Secrets stay in the VM `.env` / n8n Credentials.

n8n `/admin/workflows` is under your name (top right). Iframe of the editor. Do not treat it as the desk.

## Chat prefixes (not this website)

Do not give each department its own WhatsApp. One ops Telegram, one Academy Telegram, one customer WhatsApp, one `info@`.

| Starts with | Worker |
| --- | --- |
| `/accounts` | accounts (ops Telegram, if that desk is ticked) |
| `/ops` | internal-ops (ops Telegram) |
| `/legal` | legal (ops Telegram) |
| `/seo` | seo (ops Telegram) |
| `/file` | files (ops Telegram; Accept on Files) |
| `/today` `/pm` `/idea` | delivery (ops Telegram) |
| `/growth` | growth |
| `/crm` | crm |
| `/cm` `/community` | community |
| `/validate` | product-dev (ops Telegram) |
| none, 1:1 ops Telegram | that person’s default desk |
| none, 1:1 WhatsApp | growth (customers; CRM upsert) |
| none, group | community (Academy Telegram student rooms) |

`/approve` sends the pending **info@** draft now. `/kill` or `/stop` drops that draft and stops a delayed Delivery plan. They do not Accept Legal, Decide, or SEO.

Academy bot refuses ops prefixes. Do not add the ops bot to a class group.

## Hard no

- Do not Accept by chatting.
- Do not publish from the SEO worker.
- Do not send from the Legal worker.
- Do not write Drive from the Files worker.
- Do not hire through HR.
- Do not invent a second CAC company for a project face.
- Do not put Zoho/Paystack tokens in git or in the Go worker.
