# How you run this office

This is the operator manual for [workers.themobileprof.com](https://workers.themobileprof.com/admin/). One CAC company: **TheMobileProf Technologies** (Lagos). Bets (Academy, HomeGauge, Mechazone, …) are projects, not baby companies.

The signed-in copy is **Docs → How you run this office** (`/admin/docs/howto`). Seat sheets stay under **How you fit**. Worker JSON specimens stay on each desk sheet. Vendors: [`TOOLS.md`](TOOLS.md). Channels and n8n ids: [`CHANNEL_SETUP.md`](CHANNEL_SETUP.md).

**Workers propose. A human clicks Accept. Chat never Accepts, never Places, never sends.**

---

## 1. Four doors

| Door | Who | What it is for |
| --- | --- | --- |
| **Desk** `/admin/` | Staff with a People login | Accept, Place, Cards, Defaults, Docs. Help? (bottom right) fills the same form as the page. It never Accepts. |
| **Ops Telegram** | Staff with `telegram_id` on People | Prefixes (`/growth`, `/accounts`, `/file`…) or plain text (the bot picks **one** desk). `/approve` `/kill` `/stop` only the pending **info@** draft (`/stop` also cancels a delayed Delivery plan). |
| **Customer WhatsApp** | Anyone | Sales 1:1. Files Office → CRM when there is a phone. Refuses office prefixes. Not a class group. |
| **Academy Telegram** | Students | Community host only. Do not add the ops bot to class. |

n8n is the public door for WhatsApp, email, and cron. Go holds the desk and the workers. Tokens for Zoho, Paystack, Ads, GA4, Drive live in n8n Credentials or the VM `.env` — never in git, never on the Go worker for those vendors.

---

## 2. The one rule (every desk)

1. You Ask on the desk, send a prefixed ops message, or (ops 1:1) send plain text.
2. The worker proposes (copy, a job, a draft, a week plan, an ads pack).
3. You **Accept** or **Turn away** on the desk. Typing in chat does not Accept.

Workers never: hire, send WhatsApp/Telegram/email themselves, publish a page, write Drive (except a catalog match that files itself), stamp a project **stage**, compute VAT, invent emails/phones, or spend ads from chat.

`/approve` is not Accept.

---

## 3. Lagos clocks (they run without you)

| Time | What | Where you look |
| --- | --- | --- |
| 06:00 | Organic GA4 last 28 days | Defaults → Traffic snapshots; quote only these |
| 06:10 | Search Console last 28 days | same |
| 06:20 | Paid GA4 last 7 days + funnel | Growth book. If there were paid sessions, a **proposed** glance job (one per day). Does not Place. |
| 07:00 | Ads prune | After a full week: pause cells with spend and 0 clicks; turn off wasted dayparts. Conversions wait 14 days and 5+ clicks, and only if tracking already fired. |
| 08:30 | Today on the desk | Ops Telegram: tasks, Coming up, CRM due (owner/BDM) |
| 09:00 Mon | Class intro | Mandated Academy rooms whose bet stage allows pings |
| 09:15 | Scholarship nudge | Only when a season ends within 21 days. Never posts a discount code. |
| 09:30 | Class check-in | Greeting / where-are-you / lesson question. LLM rewords. |
| 16:00 | ₦500 airtime | Even when nobody earned it. Names yesterday’s winner when there is one. ClubKonnect from the VM `.env`. |

Campaigns start **PAUSED** unless Defaults → Ads says ENABLED.

---

## 4. Run a bet (Projects)

Every bet uses the same kit. Open the project → **Run**.

1. **Card** — who pays, one-liner, this week’s line, public URL, optional GitHub repo, Funnel line from `TRAFFIC.md`.
2. **Face** — seat a cofounder (public face) and/or assistant (field week).
3. **Decide** — Ask placement, then Accept / Amend / Reject. That writes the stage. Chat never stamps GO / PIVOT / KILL.
4. **Dates** — campus week, field week, launch. Board shows Coming up.
5. **Work** — one Delivery task. Stage caps how many may be open (idea 1, testing 3, selling 5, fundable 4). Extra sits on Backlog.

Then as needed: Legal, Files, onsite SEO in Cursor, pitch deck, Search ads.

`/validate` on ops Telegram proposes Card hypothesis, mission, and evidence. Accept on **Product** writes those fields. Accept on **Decide** still writes the stage.

---

## 5. Growth

Tick Growth on People. Customer WhatsApp is already Growth (sales).

### Sales copy / replies

Ops: `/growth Write a LinkedIn post about…`  
WhatsApp 1:1: just talk. Every DM with a phone files CRM. Books contact only with an email or once stage is qualified+.

### Pitch deck

1. Card one-liner first.
2. Project → **Deck** → Start session, or `/growth Pitch deck for HomeGauge`.
3. Chat Guy Kawasaki’s ten slides. The markdown pack URL updates every turn.
4. Accept on `/admin/growth`. You paste into Slides. Export PDF, `/file` under **product**.
5. Missing facts stay `[TO BE COMPLETED]`. Do not invent MAU or a competitor grid.

### Search ads (then Place)

1. Public URL on the Card. Geo and keyword tests are named places + themes **already on the Card** — not age or gender.
2. Project → **Ads** → Start ads session, or `/growth Search ads for HomeGauge`.
3. Six steps: final URL, cells, keywords, headlines (≤30), descriptions (≤90), Pexels queries (you search; do not generate faces).
4. **Go drops** a city, keyword, or Pexels line that is not on the Card, and records that on the pack (`Looked`).
5. Accept on Growth, then **Place** (Google Search or Meta) on the Growth book. Chat never Places.
6. Place is a cheap probe: platform-minimum daily, never more than remaining/90, optional Defaults ceiling, exact match, Google Search only (no partners). One experiment per cell. UTM: `utm_campaign` = bet slug, `utm_content` = cell.
7. n8n holds Ads tokens. Ids on Defaults → Ads. `GOOGLE_ADS_DEVELOPER_TOKEN` in the VM `.env` for n8n.

### Evaluate ads (no spend)

- Growth book shows last-7-days paid sessions and the funnel.
- After 06:20, a **Paid traffic glance** job quotes those numbers. Turn away if you do not need it. It does not Place.
- Ops: `/growth How are the HomeGauge ads doing in GA4?`
- Funnel events must already fire. Prefer the Funnel line on the Card (`TRAFFIC.md`). Defaults → Ads `ads.funnel_steps` is the company-wide fallback. Do not invent `generate_lead`.

Pexels: search the phrases yourself, `/file` stills under **product**.

---

## 6. Onsite SEO and funnels (not a worker)

There is no SEO clerk. `/seo` on ops Telegram says so.

1. Put public URL + GitHub repo on the Card.
2. Open that repo in Cursor. Copy `.cursor/rules/onsite-seo.mdc` and `traffic.mdc` from this office (set `alwaysApply: true` on `traffic.mdc` in the bet repo).
3. Copy `TRAFFIC.md` to the **bet repo root** (skeleton: `agents/internal/admin/guide/TRAFFIC.md`).
4. Fill Offer, Search table (one primary phrase per URL), Funnel line, Ads landing from the **designed page**.
5. Keep `TRAFFIC.md` in the **same diff** as a new URL or `gtag` event.
6. Paste the Funnel line onto the Card (SEO tab has the same field).
7. Walk cite → FAQ → schema → title/meta from the visible page. Organization is TheMobileProf Technologies.
8. Offsite listings you already have: add a Delivery task. Do not invent ChatGPT citations.

GA4 / Search Console **ids** on Defaults → Traffic. Last-28-days snapshots from n8n (`ga4-read`). This office does not push PRs.

---

## 7. Community (Academy rooms)

One **mandate** per student Telegram room (`/admin/community`): brief, catalog, chat id. Add the Academy bot to the group.

- Students DM the **Academy** bot the email on their LMS account — never in the group. `/invite` is the LMS join page.
- Morning: 09:30 check-in, 16:00 ₦500 even when nobody earned it.
- Campus trainings belong on project **Dates**, not here.
- Customer WhatsApp cannot run `/cm`.

---

## 8. CRM

Office → CRM (`/admin/crm`): a person, an offer, a stage, a timeline.

- WhatsApp, info@, `/crm`, and a logged call file a touch.
- `/crm follow up Apex` proposes the next human step. Due **done** clears the 08:30 ping without changing stage.
- Won stays for a later upsell. It does not send.
- n8n upserts a Zoho **Books** contact when there is an email, or once stage is qualified / proposal / won. There is no Zoho CRM app.

---

## 9. Accounts and Books

Ops Telegram: send a receipt with `/accounts` (or `/ops` when it classifies as accounts).

- The clerk names the Zoho document (expense / bill / invoice). n8n posts it. Go never computes VAT.
- Confirm the audit trail in that chat (document id or Paystack link).
- Hypotheticals must say “do not record”.
- Chart-of-accounts ids live on Defaults → Zoho, not in the message.
- Customer WhatsApp cannot book this.

---

## 10. Legal

- Project → Legal → Ask draft, or `/legal` on ops Telegram. Accept on `/admin/legal`. Copy the text yourself. It cannot email the other party.
- After another desk answers on ops Telegram, Legal **watch** flags labour / CAC / NDPR / payments / tax / contract gotchas. Caution and stop also file a Legal job.
- Do not invent case citations. Customer WhatsApp `/legal` is refused.

---

## 11. Delivery and Files

**Delivery** (`/admin/delivery`): week board across bets. Ask a plan, or accept Fathom notes after a recording (calendar title should name the bet slug). Plans write after a delay unless you Turn away. Own `/today` check-ins may apply immediately.

**Files**: ops Telegram `/file` or drop a PDF. A catalog match files itself; n8n writes Drive. Odd paths wait for Accept. Office papers use slug `themobileprof`, not a bet.

---

## 12. Defaults vs secrets

| Lives on Defaults (`/admin/settings`) | Lives in VM `.env` or n8n Credentials |
| --- | --- |
| Zoho org / chart ids, Paystack bank id, Drive id | Zoho / Paystack / Drive OAuth |
| Ads customer / page ids, optional daily cap, PAUSED vs ENABLED | Google Ads OAuth, `GOOGLE_ADS_DEVELOPER_TOKEN`, Facebook Ads |
| `seo.ga4_property_id`, Search Console property, `ads.funnel_steps` fallback | Google Analytics service account JSON (`ga4-read`) |
| | ClubKonnect keys (airtime) |

Saving one Defaults tab does not wipe the others. Amend **this group**.

---

## 13. What to open when

| I want to… | Open |
| --- | --- |
| See every bet | Board (owner/BDM) |
| Who pays / this week | Project → Card |
| Change stage | Project → Decide → Accept |
| Draft NDA | Project → Legal or `/legal` |
| File a PDF | Ops `/file` |
| Titles and funnel for a site | Cursor in the bet repo + `TRAFFIC.md` + Card Funnel |
| Pitch | Project → Deck |
| Ads copy then spend | Project → Ads → Accept → **Place** |
| See if ads became sessions | Growth book GA4 + glance job |
| Class ₦500 | Mandates; 16:00 clock. Do not Ask Community to “send airtime” from WhatsApp |
| Book a receipt | Ops `/accounts` |
| Today’s work | Delivery or 08:30 Telegram card |
| Lost on a page | Help? (never Accepts) |

---

## 14. Cursor in this repo

Architecture constraints: [`CURSOR_INSTRUCTIONS.md`](CURSOR_INSTRUCTIONS.md). Office truth: [`OFFICE.md`](OFFICE.md). Do not recommend scraping, auto-send, auto-Place, or a second company per bet.
