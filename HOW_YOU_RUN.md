# How you run this office

This is the operator manual for the company desk at [workers.themobileprof.com](https://workers.themobileprof.com/admin/). The legal entity is **TheMobileProf Technologies** in Lagos. Academy, HomeGauge, Mechazone, and the others are incubated **bets** (projects) under that one company, not separate CAC entities.

When you are signed in, the same manual lives at **Docs → How you run this office** (`/admin/docs/howto`). Seat sheets (owner, BDM, project face, field week, handler) stay under **How you fit**. JSON specimens for people who run a worker sit on each desk sheet. Vendor ids and secrets are listed in [`TOOLS.md`](TOOLS.md). Channel bots, n8n workflow ids, and webhook notes are in [`CHANNEL_SETUP.md`](CHANNEL_SETUP.md).

Workers propose. A human clicks **Accept** on the desk. Chat never Accepts, never Places an ad, and never sends mail or WhatsApp on its own.

---

## 1. Four doors

Staff and customers reach this office through four doors. Do not mix them.

| Door | Who uses it | What it is for |
| --- | --- | --- |
| **Desk** at `/admin/` | Anyone with a People login | This is where you Accept jobs, Place ads, edit project Cards, and amend Defaults. The brass **Help?** button at the bottom right fills the same form as the page you are on; it never Accepts. |
| **Ops Telegram** | Staff whose People card has a `telegram_id` | Prefixes such as `/growth`, `/accounts`, and `/file` run only if that desk is ticked on your card. Plain 1:1 text lets the bot pick **one** desk and do that work in the chat. `/approve`, `/kill`, and `/stop` apply only to the pending **info@** draft; `/stop` also cancels a delayed Delivery plan. |
| **Customer WhatsApp** | Anyone who messages the Business number | A 1:1 chat is sales. When the message includes a phone, the desk files the person on Office → CRM. This number refuses office prefixes and cannot join a class or project group. |
| **Academy Telegram** | Students in a mandated class room | Community is the host. Do not add the ops bot to a student group. |

n8n is the public door for WhatsApp, inbound email, and cron. A Go process holds the desk and the workers. Tokens for Zoho, Paystack, Ads, GA4, and Drive live in n8n Credentials or the VM `.env`. They never belong in git, and the Go worker never holds those vendor tokens.

---

## 2. The one rule (every desk)

1. You Ask on the desk, you send a prefixed ops Telegram message, or (on ops 1:1) you send plain text.
2. The worker proposes: copy, a job, a draft, a week plan, or an ads pack.
3. You click **Accept** or **Turn away** on the desk. Typing in chat does not Accept.

Workers never hire anyone, never send WhatsApp, Telegram, or email themselves, never publish a web page, and never write Google Drive except when a catalog match files itself. They never stamp a project **stage**, never compute VAT, never invent emails or phones, and never spend ads from chat.

`/approve` on ops Telegram is not Accept. It only sends a pending info@ draft if one exists.

---

## 3. Lagos clocks (they run without you)

These jobs fire on Lagos time whether or not you are at the desk.

| Time | What runs | Where you read the result |
| --- | --- | --- |
| 06:00 | Organic GA4 for the last 28 days | Defaults → Traffic. Quote only this snapshot; do not invent sessions. |
| 06:10 | Search Console for the last 28 days | The same Traffic tab. |
| 06:20 | Paid GA4 for the last 7 days, plus the funnel of events that already fire | The Growth book. If there were paid sessions, Go files one **proposed** glance job for that Lagos day. The glance does not Place. |
| 07:00 | Ads prune | After a campaign has run a full week, prune pauses a cell that spent with 0 clicks and turns off Lagos dayparts that wasted while another daypart got clicks. Conversions wait 14 days and at least 5 clicks, and only if tracking already fired. |
| 08:30 | Today on the desk | Ops Telegram lists today’s Delivery tasks, Coming up dates, and (for owner and BDM) CRM due rows. |
| 09:00 Monday | Class intro | The Academy bot introduces itself in each mandated room whose linked bet stage allows pings. |
| 09:15 | Scholarship nudge | A short reminder only when a scholarship season ends within 21 days. It never posts a discount code. |
| 09:30 | Class check-in | A greeting, a where-are-you, or a lesson question. The host LLM may reword; Go keeps the facts. |
| 16:00 | ₦500 airtime | Community sends ₦500 even when nobody earned it, names yesterday’s winner when there is one, and uses ClubKonnect keys from the VM `.env`. |

New ad campaigns start **PAUSED** unless you set ENABLED on Defaults → Ads.

---

## 4. Run a bet (Projects)

Every bet uses the same base kit. Open the project and stay on the **Run** tab until the step marked **now** is done.

1. **Card** holds who pays, the one-liner, this week’s mission, the public URL, an optional GitHub repo, and the Funnel line copied from `TRAFFIC.md`.
2. **Face** seats a cofounder as the public face of the bet and, when needed, an assistant for field week.
3. **Decide** is where you Ask placement, then Accept, Amend, or Reject. That click writes the stage. Chat never stamps GO, PIVOT, or KILL.
4. **Dates** holds a campus week, a field week, or a launch until you mark it Delivered. Board shows those rows as Coming up.
5. **Work** is one Delivery task. The project’s stage caps how many tasks may be open (idea 1, testing 3, selling 5, fundable 4). Extra items sit on Backlog.

Legal drafts, Files, onsite SEO in Cursor, a pitch deck, and Search ads come after that kit, when the bet needs them.

On ops Telegram, `/validate` can propose a Card hypothesis, this week’s mission, and evidence. Accept on **Product** writes those fields onto the Card. Accept on **Decide** is still what writes the stage.

---

## 5. Growth

Tick Growth on the handler’s People card. Customer WhatsApp is already Growth for sales; it is not the pitch session and it is not Place.

### Sales copy and replies

On ops Telegram, send `/growth Write a LinkedIn post about…` when you want copy you will post yourself. On customer WhatsApp, just talk: a 1:1 message is a sales conversation. When the message includes a phone number, the desk files that person on Office → CRM. n8n upserts a Zoho Books contact only when there is an email, or once the CRM stage is qualified, proposal, or won.

### Pitch deck

Write the Card one-liner first so the clerk has facts. Open the project → **Deck** → Start session, or send `/growth Pitch deck for HomeGauge` on ops Telegram. Chat Guy Kawasaki’s ten slides from the Card; the markdown pack URL updates every turn. Accept the pack on `/admin/growth`. You paste the markdown into Slides, export a PDF, and `/file` it under **product**. Missing facts stay `[TO BE COMPLETED]`. The clerk must not invent MAU, waitlist counts, or a competitor grid.

### Search ads, then Place

Put a public URL on the Card before you start. Test cells are named places plus query themes that already appear on the Card — not age or gender.

Open the project → **Ads** → Start ads session, or send `/growth Search ads for HomeGauge`. The walk has six steps: final URL, cells, keywords, headlines (each ≤30 characters), descriptions (each ≤90), then Pexels queries a human will search. Do not generate faces.

After the clerk drafts, Go drops a city, keyword, or Pexels line whose words are not on the Card, and records that drop on the pack under **Looked**.

Accept on Growth, then click **Place** (Google Search or Meta) on the Growth book. Chat never Places. Place is a cheap probe: the platform-minimum daily budget, never more than remaining funds divided by 90, an optional ceiling on Defaults, exact match only, and Google Search only (no partner network). Each geo/keyword cell is its own experiment. Place stamps `utm_campaign` to the bet slug and `utm_content` to the cell.

n8n holds the Ads tokens. Account ids live on Defaults → Ads. `GOOGLE_ADS_DEVELOPER_TOKEN` stays in the VM `.env` for n8n, not on the Go worker.

### Evaluate ads without spending

The Growth book shows the last seven days of paid sessions and the funnel. After the 06:20 pull, a **Paid traffic glance** job quotes those numbers; Turn away if you do not need it. The glance does not Place.

On ops Telegram you can ask `/growth How are the HomeGauge ads doing in GA4?` for the same read-only note.

Funnel steps are events that already fire on the site. Prefer the Funnel line on the Card, which you paste from `TRAFFIC.md`. Defaults → Ads `ads.funnel_steps` is only the company-wide fallback. Do not invent `generate_lead` if that event is not on the Card.

Search the Pexels phrases yourself, download stills, and `/file` them under **product**.

---

## 6. Onsite SEO and funnels (not a worker)

There is no SEO clerk. `/seo` on ops Telegram says the worker is gone.

Put the public URL and GitHub repo on the Card, then open that repo in Cursor. Copy `.cursor/rules/onsite-seo.mdc` and `.cursor/rules/traffic.mdc` from this office if they are missing, and set `alwaysApply: true` on `traffic.mdc` in the bet repo. Copy `TRAFFIC.md` to the **bet repo root**; the office skeleton is `agents/internal/admin/guide/TRAFFIC.md`.

Fill Offer, the Search table (one primary phrase per URL), the Funnel line, and the ads landing from the **designed page**. Keep `TRAFFIC.md` in the **same git diff** as a new public URL or a new `gtag` event. Paste the Funnel line onto the Card (the SEO tab has the same field).

Walk cite → FAQ → schema → title and meta from the visible page. The legal name in Organization schema is TheMobileProf Technologies unless the page already names someone else. Offsite listings you already have belong on Delivery as ordinary tasks. Do not invent ChatGPT citations.

GA4 and Search Console **ids** live on Defaults → Traffic. Last-28-days snapshots arrive from n8n as `ga4-read`. This office does not push pull requests.

---

## 7. Community (Academy rooms)

Create one **mandate** per student Telegram room at `/admin/community` (brief, catalog, chat id). Add the Academy bot to the group so the `lms` chat id can bind.

Students DM the **Academy** bot the email on their LMS account; they must not paste billing in the group. `/invite` is the LMS join landing.

At 09:30 Lagos the host checks in. At 16:00 it sends ₦500 even when nobody earned it, and names yesterday’s winner when there is one.

Campus trainings belong on the project **Dates** tab, not on Community. Customer WhatsApp cannot run `/cm`.

---

## 8. CRM

Office → CRM (`/admin/crm`) is the conversion book: a person, an offer, a stage, and a timeline of email, WhatsApp, and calls.

Customer WhatsApp, inbound info@, `/crm`, and a logged call each file a touch on that person. `/crm follow up Apex` proposes the next human step from that story. Marking due **done** (or `/crm done Apex`) takes them off the 08:30 ping without changing stage.

Won stays on the book for a later upsell. CRM never sends mail or WhatsApp. n8n upserts a Zoho **Books** contact when there is an email, or once the stage is qualified, proposal, or won. There is no Zoho CRM app.

---

## 9. Accounts and Books

On ops Telegram, send a receipt with `/accounts`, or with `/ops` when the classifier routes it to accounts.

The clerk names the Zoho document (expense, bill, or invoice). n8n posts it. Go never computes VAT or withholding tax. Confirm the audit trail in that same chat (document id or a Paystack link). Hypotheticals must say “do not record”. Chart-of-accounts ids live on Defaults → Zoho, not in the message. Customer WhatsApp cannot book this.

---

## 10. Legal

On the project, open Legal → Ask draft, or send `/legal` on ops Telegram. Accept the proposal on `/admin/legal`, then copy the text yourself. The clerk cannot email the other party.

After another desk answers on ops Telegram, Legal **watch** flags labour, CAC, NDPR, payments, tax process, or contract gotchas. A caution or stop also files a Legal job. Do not invent case citations. Customer WhatsApp `/legal` is refused.

---

## 11. Delivery and Files

**Delivery** (`/admin/delivery`) is the week board across every bet. You add a task, Ask a plan, or accept Fathom notes after a recording (put the bet slug in the calendar title). Plans write after a delay unless you Turn away. Your own `/today` check-ins may apply immediately.

**Files** is the data room. On ops Telegram send `/file` or drop a PDF. A catalog match files itself and n8n writes Google Drive. Odd paths wait here for Accept. Office papers use the slug `themobileprof`, not a bet name.

---

## 12. Defaults versus secrets

| Lives on Defaults (`/admin/settings`) | Lives in the VM `.env` or n8n Credentials |
| --- | --- |
| Zoho org and chart-of-accounts ids, Paystack bank id, Shared Drive id | Zoho, Paystack, and Drive OAuth tokens |
| Ads customer and Page ids, optional daily cap, PAUSED or ENABLED | Google Ads OAuth, `GOOGLE_ADS_DEVELOPER_TOKEN`, Facebook Ads |
| `seo.ga4_property_id`, Search Console property, company-wide `ads.funnel_steps` fallback | Google Analytics service account JSON (`ga4-read`) |
| | ClubKonnect keys for airtime |

Saving one Defaults tab does not wipe the others. Use **Amend this group** on that vendor only.

---

## 13. What to open when

| If you want to… | Open |
| --- | --- |
| See every bet | Board (owner and BDM) |
| Read who pays and this week’s line | The project **Card** |
| Change the stage | Project → **Decide** → Accept |
| Draft an NDA | Project → Legal, or ops `/legal` |
| File a PDF | Ops Telegram `/file` |
| Titles, phrases, and the ads funnel for a public site | Cursor in the bet repo, `TRAFFIC.md`, then the Funnel field on Card |
| Walk a pitch | Project → **Deck** |
| Draft ads and then spend | Project → **Ads** → Accept → **Place** |
| See whether ads became sessions | The GA4 block on the Growth book, or the morning glance job |
| Send class ₦500 | Mandates plus the 16:00 clock. Do not Ask Community from customer WhatsApp to “send airtime” |
| Book a receipt | Ops Telegram `/accounts` |
| See today’s work | Delivery, or the 08:30 Telegram card |
| Get unstuck on a page | Help? (it never Accepts) |

---

## 14. Cursor in this repo

Architecture constraints live in [`CURSOR_INSTRUCTIONS.md`](CURSOR_INSTRUCTIONS.md). The live-system description is [`OFFICE.md`](OFFICE.md). Do not recommend scraping the public web, auto-send, auto-Place, or a second company per bet.
