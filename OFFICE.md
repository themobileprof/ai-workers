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
- send WhatsApp, Telegram, or email themselves (Growth drafts; info@ sends after a delay unless `/stop`; `/approve` sends that draft now)
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
- **People** — roster, hours/week, desk ticks, WhatsApp allowlist.
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
| Ops Telegram | Private office bot. Prefixes. `/file` for the data room. |
| Customer WhatsApp | One 1:1 number. Bare message is sales (Growth). Books/legal/SEO only if the phone is allowlisted. Cannot join ordinary groups. |
| Academy Telegram | Student class rooms only. Not the ops bot. |
| info@ | Formal mail. Growth drafts. Sends after a delay unless `/stop`. `/approve` sends now. |
| Fathom | Meeting notetaker. Webhook files a **proposed** Delivery plan. Accept on Job chat. Calendar title should name the bet slug. |

---

## 6. Workers that exist (keep these jobs)

Do not merge these away. Do not rename them to Silicon Valley roles.

### Accounts (`/accounts`) — live

Classifier. Names the Zoho document from text or a receipt photo. n8n posts to Zoho Books and Paystack. Lookup/cash P&L is a Books read.

It is **not** a comptroller, not monthly close, not VAT math. A monthly NGN figure you paste lives on the project Card. Books tracking categories come later.

### Ops (`/ops`) — partial

Wider prompt: accounts, legal-read, grant eligibility. Accounts via `/ops` still writes Books. `/ops` legal does **not** file a draft (use Legal). Grant hunting writes a deadline row on **Grants**. It does not submit an application.

### Growth (`/growth`) — live

Inbound front office: customer WhatsApp, info@ drafts, social copy. High intent may upsert a Zoho Books contact.

It is **not** an outbound scraper. It cannot send. Bare 1:1 WhatsApp is Growth.

### CRM (`/crm`) — live

Qualify or update a person. n8n writes **Zoho Books contacts**. There is no Zoho CRM app.

It is **not** a cold-outreach sequencer.

### Community (`/cm`) — live — keep

**Academy Telegram class rooms.** One mandate per group (brief + LMS catalog). Monday intro when the linked bet’s stage allows pings.

It is **not** an AI interviewing early customers of HomeGauge or Mechazone. Field week already does that with humans. Do not remove Community “until Stage 4.” Class still needs a host.

### Legal (`/legal`) — partial — keep

Parent clerk for TheMobileProf: proposed NDAs and contracts from git templates. Humans Accept on the Legal book. Copy the text yourself. Telegram **watch** flags labour/CAC/NDPR/CBN on ops threads; it does not save a row and does not halt Books.

It is **not** a grant officer. Grants live on Office → Grants (`/ops` can file a row). Do not delete Legal because grants exist. Faces still need paper.

### SEO (`/seo`) — partial

Titles, meta, outlines, audit punch lists. Accept, then a human pastes onto the live page.

It cannot publish, crawl, or read Search Console. It is **not** GEO / “win AI answers.” It is not new.

### Files (`/file`) — partial

Ops Telegram `/file` or a dropped PDF. Clerk picks a catalog folder. A catalog match files itself. Unsure rows stay proposed for Accept. n8n writes a Google Shared Drive.

The worker cannot upload or invent a folder.

### Delivery (`/today`) — live — keep

**Project PM across every bet.** Not GitHub Actions, not Vercel, not UptimeRobot.

Ask plan proposes a week. It writes after a delay unless you Turn away. Extra items sit on Backlog. Fathom (webhook `/webhook/pm-notes`) files the same kind of proposed plan from a finished call. Own `/today done` may apply immediately. Stage and hours/week still cap work. Seat on Face to onboard (that is not hiring). Do not nag this on customer WhatsApp.

If you remove Delivery, field people have nowhere to be given work.

### Product-dev (`/validate`) — partial

Engineering lab: falsifiable hypotheses, intern **missions**, `analyse_evidence`, **journey placement**. The Card stores hypothesis and this week’s mission. Ask placement on the project; Accept on **Decide** writes the stage. WhatsApp a mission to a field phone is later.

It is **not** a ticket mill and not a code agent. Delivery is the backlog. Do not simplify Product to “only tag transcripts” — that deletes placement, which is the incubator.

Journeys in git (examples): consumer subscription, Academy catalog, invited AUM, mortgage enablement, shop-floor ledger. Gates have a Done-when and a default mission. The worker may only pick IDs from that catalog.

---

## 7. What is later (real gaps)

These are on the desk now: grant rows, Delivery backlog, delay-to-send info@, delay-to-write week plans, catalog-matched Files, hypothesis/mission/spend on the Card, Board rollup.

Still later:

1. WhatsApp a mission to a field phone (do not use customer WhatsApp). GO / PIVOT / KILL stays Accept on Decide.
2. Burn-by-bet **from Books tracking categories** (Card can hold a monthly NGN figure you paste; Go still does not compute VAT).
3. Search Console later — still no publish.

Community stays Telegram rooms, not a campus timetable.

---

## 8. What not to recommend

Do not tell the owner to:

1. Remove Legal, Delivery, or Community.
2. Merge Delivery into “Product/Ops” because “DevOps tools already do that.” Delivery is a week board for humans.
3. Turn Growth into scrape + cold sequences.
4. Add a Scout that scrapes Reddit, X, or forums to “see if the problem is growing.” Proof here is a **named person at a named place**.
5. Add a Demo Day worker that invents waitlist / MAU / burn per “startup.” Those metrics are not stored. Copy can be drafted by Growth; a human sends.
6. Add GEO, crawlers, or programmatic publishing.
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
| `/accounts` | accounts (WhatsApp allowlisted) |
| `/ops` | ops (WhatsApp allowlisted) |
| `/legal` | legal (WhatsApp allowlisted) |
| `/seo` | seo (WhatsApp allowlisted) |
| `/file` | files (ops Telegram) |
| `/today` `/pm` `/idea` | delivery |
| `/growth` | growth |
| `/crm` | crm |
| `/cm` `/community` | community |
| `/validate` | product-dev |
| none, 1:1 | growth |
| none, group | community (Academy student rooms) |

---

When advising this office: keep the live books, keep the Accept rule, keep field humans, and only extend the “later” list in section 7.
