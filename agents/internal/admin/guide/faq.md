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

## What is Stamp? What does Decide mean?

Stamp is not a badge you collect. It is an old word for **your yes**.

A worker can only propose. The coloured label on a card (Idea, Testing, Selling…) is the **stage**. The stage does not change until you click **Accept**.

1. Open the project → **Decide** (the first tab).
2. Read the proposal. Click **Accept** if you agree — that yes is the stamp.
3. Click **Reject** to leave the stage as it is. Chat, Help?, and Telegram `/approve` never do this.

The stage then caps how many Delivery tasks that project may hold: idea 1, testing 3, selling 5, fundable 4, parked/killed 0.

On Legal, SEO, and a worker desk the same click is **Accept** or **Turn away**.

## Can Help? fill a form for me?

Yes. You can type on the page, or talk to **Help?** (bottom right — it used to say How?) — same result. The guide reads your intent (LLM) and picks a form from the desk catalog, then asks for each missing box. **Submit** posts that form. It never clicks **Accept**. New forms join the catalog; there is no keyword list to reprogram.

1. Open **Help?** (bottom right). Tap a chip or say what you need done.
2. Answer the next box it asks. Say **cancel** to stop.
3. Check the summary. Click **Submit** (or **Open form** and type it yourself).

## What is the difference between Help? and Ask?

**Help?** (bottom right) explains a button **or** walks the same form as the page. Submit in Help? posts that form. Ask on a worker desk is one of those forms — it files a **proposed** job. Chat a fix if needed. **Accept** or **Turn away** is still your yes or no. Help? never Accepts.

## How do I assign work this week?

1. Seat the person on the project Face tab (unless they are owner or BDM).
2. Open Delivery (`/admin/delivery`) → Add a task by hand, or Ask plan then Accept on Job chat.
3. Or WhatsApp `/today` from a People phone. How many tasks you may hold depends on the project’s stage: idea 1, testing 3, selling 5, fundable 4, parked/killed 0. Hours/week on People is the ceiling.

## How do I onboard someone onto a bet?

1. People → their card: set hours/week.
2. Project → Face: pick them, seat cofounder or assistant, click Seat them.
3. Seat is not a login. Login is the desk tick on People.

## What does /approve do?

On ops Telegram it only sends the pending info@ email draft. `/kill` drops that draft. It does not Accept a contract, SEO brief, delivery plan, project gate, or hire. Recruiting is closed.

## How do I talk to the Academy / community bot?

Student rooms are Academy Telegram, not WhatsApp groups (Cloud API cannot join ordinary groups). Owner sets a mandate on `/admin/community` (brief, catalog, `telegram_chat_id`). `/cm` or `/community` on the Academy bot. Ops prefixes on that bot are refused. Do not add the ops bot to class.

## How do I file an NDA?

Tick Legal on the handler. Workers → Legal → Ask for the draft you need. Wait for the proposed job. Chat a fix. Accept on the job or on `/admin/legal`. Copy the body from the Legal book if you need the document. The worker cannot send it.

## How do I ask for SEO copy?

Tick SEO. Workers → SEO → Ask, or `/seo` from an allowlisted WhatsApp number. Accept on `/admin/seo`. Copy from the desk. Publishing the live page is not wired. The worker cannot crawl or read Search Console.

## Why can’t I see the Board?

Board is owner and BDM. Handlers land on Workers. Project faces should not see the whole office. If you should be BDM, ask the owner to change your People seat.

## How do I change Zoho or Paystack ids?

1. Open Defaults (`/admin/settings`) → Amend ids.
2. Paste the Zoho Books account numbers. These are ids, not secrets.
3. Save. API tokens stay in the VM `.env` or n8n Credentials — never on this page.

## Someone sent a CV. Do we hire them in HR?

No. Recruiting is closed. Onboard people you already know via People + Face. Inbound CVs go to Growth as a conversation, not a hire.

## How do I mark a delivery task done?

1. Open Delivery (`/admin/delivery`).
2. Click **Done** on the task (or **Skip**).
3. Or WhatsApp `/today done …` from your own People phone. Do not use `/approve`.

## What should a field person do?

1. Read this week’s line on the project card. That is the mission.
2. Go to the place. Bring named people back.
3. You decide GO / PIVOT / KILL on the project. They often have no login. They are not a director of TheMobileProf.

## How do I track a training, campus week, or dated event on a project?

1. Open the project → **Dates**.
2. Fill What, start date, Where, Who, and success. Leave “Set as this week’s line” ticked if this is the agenda. Click **Save date**.
3. Click **Add task on Delivery** for the work. After it happened, click **Delivered**.

Seat campus helpers on Face as assistant. Do not put this in Community (those are Telegram student rooms). Do not use `/approve`.
