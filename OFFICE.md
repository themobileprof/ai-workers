# TheMobileProf office — source of truth

Use this file as the description of the live system. It is not a pitch. It is not a generic “solo incubator AI stack.” If this file and a brochure disagree, this file wins.

**Company:** TheMobileProf Technologies (one CAC company, Lagos).  
**Desk:** https://workers.themobileprof.com/admin/  
**Rule:** Workers propose. A human clicks **Accept**. Chat never Accepts.

Do not recommend scraping the public web, cold outreach that sends itself, publishing pages, sending contracts, writing Google Drive, hiring anyone, or inventing a second company per project.

---

## 1. What this is

This is one Lagos office that incubates several **bets** (projects) under one legal entity. Every invoice, NDA, and Books contact is TheMobileProf Technologies. Academy LMS, Mechazone, HomeGauge, and the others are bets, not baby companies.

n8n is the public door (WhatsApp, Telegram, email). A Go process holds the workers and the **desk** (the website). Humans sign in on the desk. That is where Accept lives.

A generic six-worker “comptroller / outbound engine / GEO / ticket mill / grant officer / meeting secretary” list describes a different product. Do not reshape this office to match it.

---

## 2. The one rule (never violate)

1. A human Asks, or sends a prefixed chat.
2. The worker proposes (text, a job, a draft, a path, a week plan).
3. A human **Accepts** or **Turns away** on the desk. Typing in chat does not Accept.

Workers never:

- hire anyone (recruiting is closed)
- send WhatsApp, Telegram, or email themselves (Growth drafts only when a human asks; inbound info@ files a story and does not send; `/approve` sends a pending draft if one exists)
- publish a web page
- write Google Drive (a catalog match files itself; unsure rows stay proposed until Accept)
- stamp a project **stage** (that is Accept on **Decide**)
- compute VAT or WHT (Zoho Books does)
- invent emails, phones, grant receipts, or case citations

`/approve`, `/kill`, and `/stop` on ops Telegram only send or drop the pending **info@** draft (and `/stop` also cancels a delayed Delivery plan). They do not Accept legal, SEO, hiring, or a project stage.

---

## 3. People are seats, not departments

| Seat | What they do |
| --- | --- |
| Owner | Runs the office. Adds people. Sees every bet. Clicks Accept. |
| BDM (business desk) | Same Board. Accepts on bets. Cannot add people. |
| Project face | Seated **cofounder** on one bet. Public face of that bet. Not a director of TheMobileProf. |
| Field week | Seated **assistant**. Goes to a place, brings **named people** back. Often has no login. |
| Handler | Ticked on a worker desk. Ask → chat → Accept. |

Login is a People card with a password and at least one desk tick. A Face seat is not a login.

Customer development is a **human** on Face with this week’s line (“three Ikeja mortgage desks, today’s rate on letterhead or a named officer”). It is not a scraper and not an AI in the room.

---

## 4. Desk OS (not a worker)

These are the website, not extra agents:

- **Board** — owner/BDM. Cards, Coming up dates, proposal waiting, rollup (tasks, files, grants, spend).
- **Projects** — bets. Tab **Run** is the wizard: same base kit for every bet. Card holds hypothesis, this week’s mission, and a monthly NGN figure you paste.
- **People** — roster, hours/week, desk ticks, Telegram id (ops bot).
- **Delivery** — week board across bets (this is PM, not DevOps). Extra work sits on Backlog. Plans write after a delay unless you Turn away.
- **Defaults** — one form per vendor tab (Company, Zoho, Paystack, Drive). Secrets stay in `.env` / n8n.
- **Files** — data room. Catalog match files itself; unsure stays proposed.
- **Grants** — deadline and next action. Does not submit an application.
- **Docs** — printable sheets. Start at How you fit.
- **Help?** — bottom-right guide. Fills the same forms as the page. Never Accepts.

**Run** (on each project), in order:

1. Write the card (who pays, this week’s line)
2. Seat a face
3. Place on a journey (Ask placement, then Accept on Decide)
4. Put a date on it (campus week, field week, launch)
5. Give someone work (one Delivery task)

Then: this gate’s mission from the journey catalog. Then Legal / Files / SEO as needed.

Stages (the coloured label): Idea, Testing, Selling, Fundable, Parked, Killed. Stage caps how many Delivery tasks may be open: parked/killed 0, idea 1, testing 3, selling 5, fundable 4. Hours/week on People is pace.

---

## 5. Channels (do not invent more)

Do not give each worker its own WhatsApp.

| Channel | Use |
| --- | --- |
| Desk website | Accept, Run, Dates, People, long work |
| Ops Telegram | Private office bot. Staff identified by People Telegram id. Prefixes only within that seat. `/file` for the data room. |
| Customer WhatsApp | One 1:1 number. Bare message is sales (Growth). Every customer DM with a phone upserts CRM. Books/legal/delivery stay on Telegram. Cannot join ordinary groups. |
| Academy Telegram | Student class rooms only. Not the ops bot. |
| info@ | Formal mail. CRM files a thread summary. Important pings owner/BDM (and the matching cofounder). Nothing is auto-sent. `/approve` only if a human asked for a draft. |
| Fathom | Meeting notetaker. Webhook files a **proposed** Delivery plan. Accept on Job chat. Calendar title should name the bet slug. A known CRM invitee (not staff) also gets a call event — no send, no new contact. |

---

## 6. Workers that exist (keep these jobs)

Do not merge these away. Do not rename them to Silicon Valley roles.

### Accounts (`/accounts`) — live

Classifier. Names the Zoho document from text or a receipt photo. n8n posts to Zoho Books and Paystack. Lookup/cash P&L is a Books read.

It is **not** a comptroller, not monthly close, not VAT math. A monthly NGN figure you paste lives on the project Card. Books tracking categories come later.

### Ops (`/ops`) — partial

Wider prompt: accounts, legal-read, grant eligibility. Accounts via `/ops` still writes Books. `/ops` legal does **not** file a draft (use Legal). Grant hunting writes a deadline row on **Grants**. It does not submit an application.

### Growth (`/growth`) — live

Inbound front office: customer WhatsApp and social copy. Every customer WhatsApp DM with a phone files **Office → CRM**. n8n upserts a Zoho Books contact only with an email or once the stage is qualified+.

It is **not** an outbound scraper. It cannot send. Bare 1:1 WhatsApp is Growth. Inbound info@ is CRM, not a Growth auto-reply.

### CRM (`/crm`) — live

A first-party book on the desk (**Office → CRM**). This is where conversion lives: a person, an offer, a stage, and a timeline of email, WhatsApp, and calls. WhatsApp, info@, `/crm`, and a logged call file a touch on that person. Owner and BDM see every row; a cofounder sees contacts on their bet. Important mail still pings owner and BDM. Won stays on the book for a later upsell. n8n upserts a **Zoho Books contact** when there is an email, or once the stage is qualified, proposal, or won. There is no Zoho CRM app.

It is **not** a cold-outreach sequencer. It does not send mail or WhatsApp. `/crm follow up Apex` proposes the next touch from the timeline. The 08:30 Lagos Telegram card lists due dates to owner/BDM with Delivery and Coming up. Notification sequences and upsell plays come later and still need a human.

### Community (`/cm`) — live — keep

**Academy Telegram class rooms.** One mandate per group (brief + LMS catalog). Monday intro when the linked bet’s stage allows pings.

It is **not** an AI interviewing early customers of HomeGauge or Mechazone. Field week already does that with humans. Do not remove Community “until Stage 4.” Class still needs a host.

### Legal (`/legal`) — live — keep

Parent clerk for TheMobileProf: proposed NDAs and contracts from git templates. Humans Accept on the Legal book. Copy the text yourself. Telegram **watch** flags labour/CAC/NDPR/CBN on ops threads after another worker answers. A caution or stop also files a Legal job. It does not halt Books. It does not email the other party.

It is **not** a grant officer. Grants live on Office → Grants (`/ops` can file a row). Do not delete Legal because grants exist. Faces still need paper.

### SEO (`/seo`) — partial

Titles, meta, outlines, keyword inventory from the bet’s own pages, and a walk: quotable facts, FAQ, schema, an onsite pack for Cursor, then offsite. Give a live URL. The clerk fetches that origin (JSON-LD is kept). The visible page is the source of truth (a designer may never have seen title/meta). Ask issues a markdown pack (`/site/seo/{token}`) — download it or reference the URL in Cursor. A human implements onsite in the site repo. Offsite work is handed to Delivery as a week plan (spread over periods, stage caps, backlog) that writes after the delay unless you Turn away. The clerk cannot send. It is still not a GEO desk — no invented ChatGPT citations.

Search Console and GA4 **property ids** sit on Defaults (free APIs, *your* site only). OAuth is later — do not invent query numbers until then. Competitor keyword spy tools are paid and not wired.

It cannot publish or merge. It does not talk to GitHub. It is **not** GEO / “win AI answers.” It is not a web-wide crawler.

### Files (`/file`) — partial

Ops Telegram `/file` or a dropped PDF. Clerk picks a catalog folder. A catalog match files itself. Unsure rows stay proposed for Accept. n8n writes a Google Shared Drive.

The worker cannot upload or invent a folder.

### Delivery (`/today`) — live — keep

**Project PM across every bet.** Not GitHub Actions, not Vercel, not UptimeRobot.

Ask plan proposes a week. It writes after a delay unless you Turn away. Extra items sit on Backlog. Fathom (webhook `/webhook/pm-notes`) files the same kind of proposed plan from a finished call. If the title names a bet and an invitee is already on Office → CRM (not staff), a call event is filed — no send, no new contact. Own `/today done` may apply immediately. Stage and hours/week still cap work. Seat on Face to onboard (that is not hiring). Do not nag this on customer WhatsApp. The 08:30 Lagos Telegram card is one ping: tasks, Coming up, and CRM due for owner/BDM.

If you remove Delivery, field people have nowhere to be given work.

### Product-dev (`/validate`) — live — keep

Engineering lab: falsifiable hypotheses, intern **missions**, `analyse_evidence`, **journey placement**. `/validate` proposes Card hypothesis, mission, and evidence. Accept on Product writes the Card. Ask placement on the project; Accept on **Decide** writes the stage. Chat never stamps GO / PIVOT / KILL. WhatsApp a mission to a field phone is later.

It is **not** a ticket mill and not a code agent. Delivery is the backlog. Do not simplify Product to “only tag transcripts” — that deletes placement, which is the incubator.

Journeys in git (examples): consumer subscription, Academy catalog, invited AUM, mortgage enablement, shop-floor ledger. Gates have a Done-when and a default mission. The worker may only pick IDs from that catalog.

---

## 7. What is later (real gaps)

These are on the desk now: grant rows, the CRM book (person + offer + timeline of email / WhatsApp / calls), Delivery backlog, delay-to-write week plans, catalog-matched Files, hypothesis/mission/spend on the Card, Board rollup. Inbound info@ files a story on CRM; it does not auto-send. `/approve` still sends a pending draft a human asked for.

Still later:

1. WhatsApp a mission to a field phone (do not use customer WhatsApp). GO / PIVOT / KILL stays Accept on Decide.
2. Burn-by-bet **from Books tracking categories** (Card can hold a monthly NGN figure you paste; Go still does not compute VAT).
3. CRM notification sequences (email / WhatsApp) and upsell plays, fired from the person timeline. Due-today Telegram is live. A human still Accepts or `/approve`. Do not auto-text the customer.

Community stays Telegram rooms, not a campus timetable.

---

## 8. What not to recommend

Do not tell the owner to:

1. Remove Legal, Delivery, or Community.
2. Merge Delivery into “Product/Ops” because “DevOps tools already do that.” Delivery is a week board for humans.
3. Turn Growth into scrape + cold sequences.
4. Add a Scout that scrapes Reddit, X, or forums to “see if the problem is growing.” Proof here is a **named person at a named place**.
5. Add a Demo Day worker that invents waitlist / MAU / burn per “startup.” Those metrics are not stored. Copy can be drafted by Growth; a human sends.
6. Add GEO, crawl the open web, or auto-merge / auto-publish. Fetching the bet’s own URL and writing a pack a human (or Cursor) implements is the SEO path.
7. Let any worker Accept, send, publish, hire, or file at CAC.
8. Give each bet its own WhatsApp, Books org, or CAC.

---

## 9. How a week actually runs

1. Owner/BDM opens Board. If a card says proposal waiting → Decide → Accept or Reject.
2. Open the bet → **Run**. Do the step marked now.
3. If the gate needs the street: write this week’s line, seat an assistant, they bring names.
4. Optional: `/validate` analyse that evidence. Still a proposal. Human Accepts the stage.
5. Delivery: Done on today’s tasks, or Ask plan then Accept. Fathom notes file a proposed plan the same way — Accept on Job chat. Calendar titles should name the bet slug. Do not chase this on customer WhatsApp. `/pm` on ops Telegram still pastes notes by hand.
6. Receipts: `/accounts` or `/ops`. Confirm what Books posted.
7. Paper: Ask draft on Legal, Accept, copy the text.
8. Class: Community mandate on Academy Telegram — not the ops bot.

---

## 10. Chat prefixes (quick list)

| Starts with | Worker |
| --- | --- |
| `/accounts` | accounts (ops Telegram, role-gated) |
| `/ops` | ops (ops Telegram) |
| `/legal` | legal (ops Telegram) |
| `/seo` | seo (ops Telegram) |
| `/file` | files (ops Telegram) |
| `/today` `/pm` `/idea` | delivery (ops Telegram) |
| `/growth` | growth |
| `/crm` | crm |
| `/cm` `/community` | community |
| `/validate` | product-dev (ops Telegram) |
| none, 1:1 ops Telegram | that person’s default desk |
| none, 1:1 WhatsApp | growth (customers; CRM) |
| none, group | community (Academy student rooms) |

---

When advising this office: keep the live books, keep the Accept rule, keep field humans, and only extend the “later” list in section 7.
