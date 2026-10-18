# FAQ

Durable answers. Add a `##` question when real people ask something this file does not cover. Live asks sit in Postgres until someone folds them here.

## How do I add someone to the office?

Owner only. Open People (`/admin/users`) → Add. Name, phone in `234…` form, office seat, hours/week, then tick the desks they will run. That tick is their login. Seat them on a project Face tab (cofounder or assistant) to onboard them onto a bet. BDM cannot add people.

## Why do I get 403 on Legal or SEO?

Those books require the matching desk tick on your People card. Owner: open People → your card → Desks → tick Legal and/or SEO. Delivery is already on the Board rail for owner/BDM.

## What is the difference between How? and Ask?

How? (bottom right) explains which screen to use. It does not file a job. Ask on a worker desk or book files a proposed job. Chat a fix if needed. Accept or Turn away stamps. Chat never stamps.

## How do I assign work this week?

Delivery (`/admin/delivery`) or `/today` from an allowlisted phone. Stamp on the project caps how many open tasks (idea 1, testing 3, selling 5, fundable 4, parked/killed 0). Pace is hours/week on the People card. Seat the person on the Face tab first. The worker proposes; you Accept.

## How do I onboard someone onto a bet?

People card: hours/week. Project card: Face tab, cofounder or assistant. Seat is not a login. Login is the desk tick on People.

## What does /approve do?

On ops Telegram it only sends the pending info@ email draft. `/kill` drops that draft. It does not Accept a contract, SEO brief, delivery plan, project gate, or hire. Recruiting is closed.

## How do I talk to the Academy / community bot?

Student rooms are Academy Telegram, not WhatsApp groups (Cloud API cannot join ordinary groups). Owner sets a mandate on `/admin/community` (brief, catalog, `telegram_chat_id`). `/cm` or `/community` on the Academy bot. Ops prefixes on that bot are refused. Do not add the ops bot to class.

## How do I file an NDA?

Tick Legal on the handler. Workers → Legal → Ask “Draft an NDA for Ada on Mechazone.” Wait for the proposed job. Chat a fix. Accept on the job or on `/admin/legal`. Copy the body from the Legal book if you need the document. The worker cannot send it.

## How do I ask for SEO copy?

Tick SEO. Workers → SEO → Ask, or `/seo` from an allowlisted WhatsApp number. Accept on `/admin/seo`. Copy from the desk. Publishing the live page is not wired. The worker cannot crawl or read Search Console.

## Why can’t I see the Board?

Board is owner and BDM. Handlers land on Workers. Project faces should not see the whole office. If you should be BDM, ask the owner to change your People seat.

## How do I change Zoho or Paystack ids?

Defaults (`/admin/settings`), owner/BDM. Those are Books account ids, not secrets. API tokens stay in the VM `.env` or n8n Credentials.

## Someone sent a CV. Do we hire them in HR?

No. Recruiting is closed. Onboard people you already know via People + Face. Inbound CVs go to Growth as a conversation, not a hire stamp.

## How do I mark a delivery task done?

Open Delivery (`/admin/delivery`) and stamp the item done (or skip). `/today` check-ins from your own People phone may apply immediately. Do not use `/approve`.

## What should a field person do?

They often have no login. The this-week line on the project card is the mission. They go to the place and bring named people back. The desk stamps GO/PIVOT/KILL. They are not a director of TheMobileProf.

## How do I track a training, campus week, or dated event on a project?

Open the bet (`/admin/projects` → TheMobileProf Academy). Open the **Dates** tab. Name it (e.g. Linux on Mobile — UNILAG), set 17–19 Nov 2026, venue, audience, and what success looks like (sign-ups for lms.themobileprof.com). Tick “Set as this week’s line” so Board keeps the agenda. Stamp **Delivered** when it happened.

Then file the work: on that date, “Add task on Delivery” (or Delivery itself). The project stamp still caps open tasks. Seat campus helpers on Face as assistant.

Do not invent a second company. Do not put the event only in Community mandates (those are Academy Telegram student rooms). Do not use `/approve`. The intern-mission loop is still later — Dates + Delivery is the memory.
