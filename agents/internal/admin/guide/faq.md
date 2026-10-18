# FAQ

Durable answers. Add a `##` question when real people ask something this file does not cover. Live asks sit in Postgres until someone folds them here.

## How do I add someone to the office?

Owner only (BDM cannot add people):

1. Open People (`/admin/users`) → Add person.
2. Person tab: name, phone in `234…` form, a login they will remember, hours/week (6 is fine).
3. Desks tab: tick only the workers they will run. That tick is their login.
4. Click Add. Then open the project → Face → seat them as cofounder or assistant.

## Why do I get 403 on Legal or SEO?

1. Ask the owner to open People → your card → Desks.
2. Tick Legal and/or SEO.
3. Sign out and sign in again. Delivery is already on the Board rail for owner/BDM.

## What is the difference between How? and Ask?

How? (bottom right) explains which screen to use. It does not file a job. Ask on a worker desk or book files a proposed job. Chat a fix if needed. Accept or Turn away stamps. Chat never stamps.

## How do I assign work this week?

1. Seat the person on the project Face tab (unless they are owner or BDM).
2. Open Delivery (`/admin/delivery`) → Add a task by hand, or Ask plan then Accept on Job chat.
3. Or WhatsApp `/today` from a People phone. Stamp caps: idea 1, testing 3, selling 5, fundable 4, parked/killed 0. Hours/week on People is the ceiling.

## How do I onboard someone onto a bet?

1. People → their card: set hours/week.
2. Project → Face: pick them, seat cofounder or assistant, click Seat them.
3. Seat is not a login. Login is the desk tick on People.

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

1. Open Defaults (`/admin/settings`) → Amend ids.
2. Paste the Zoho Books account numbers. These are ids, not secrets.
3. Save. API tokens stay in the VM `.env` or n8n Credentials — never on this page.

## Someone sent a CV. Do we hire them in HR?

No. Recruiting is closed. Onboard people you already know via People + Face. Inbound CVs go to Growth as a conversation, not a hire stamp.

## How do I mark a delivery task done?

1. Open Delivery (`/admin/delivery`).
2. Click **Done** on the task (or **Skip**).
3. Or WhatsApp `/today done …` from your own People phone. Do not use `/approve`.

## What should a field person do?

1. Read this week’s line on the project card. That is the mission.
2. Go to the place. Bring named people back.
3. The desk stamps GO / PIVOT / KILL. They often have no login. They are not a director of TheMobileProf.

## How do I track a training, campus week, or dated event on a project?

1. Open Projects → TheMobileProf Academy → **Dates**.
2. Fill What (e.g. Linux on Mobile — UNILAG), start/end (17–19 Nov 2026), Where, Who, and success (sign-ups for lms.themobileprof.com). Leave “Set as this week’s line” ticked. Click **Save date**.
3. Click **Add task on Delivery** for the work. After the week, click **Delivered**.

Seat campus helpers on Face as assistant. Do not put this in Community (those are Telegram student rooms). Do not use `/approve`.
