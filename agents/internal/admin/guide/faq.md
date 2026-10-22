# FAQ

Durable answers. Add a `##` question when real people ask something this file does not cover. Live asks sit in Postgres until someone folds them here.

## How do I start or run a project?

Every bet uses the same base kit. **Run** walks it in order.

1. Open the project (or **Open a project**, then you land on Run).
2. Do the step marked **now**: Card (who pays + this week), Face, Decide (Ask placement, then Accept), Dates, Work.
3. After those, Run shows this gate’s mission, then Legal / Files / SEO as needed. **Help? walk this** fills the same form. It never Accepts.

Campus trainings are Dates, not Community. Chat never stamps the stage.

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

## How do I stop a delayed email or week plan?

info@ drafts and Delivery plans wait 15 minutes (60 after 20:00 Africa/Lagos), then they go unless you stop them.

1. Ops Telegram: **`/stop`** or **`/kill`** drops the pending info@ draft and stops a delayed Delivery plan.
2. **`/approve`** still sends the info@ draft now.
3. Job chat **Turn away** stops a plan. **Accept** writes it now.

Stage, Legal send, and money above a cap stay a human click. Catalog-matched Files place themselves.

## How do I assign work this week?

1. Seat the person on the project Face tab (unless they are owner or BDM).
2. Open Delivery (`/admin/delivery`) → Add a task by hand, or Ask plan. Extra tasks sit on **Backlog** when the stage is full — Promote when a slot opens.
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

Each vendor has its own form. Saving one tab does not wipe the others.

1. Open Defaults (`/admin/settings`) → the tab for that vendor (Company, Zoho, Paystack, Drive).
2. Click **Amend this group**. Paste only those fields. These are ids, not secrets.
3. Click **Save this group**. API tokens stay in the VM `.env` or n8n Credentials — never on this page.

## Someone sent a CV. Do we hire them in HR?

No. Recruiting is closed. Onboard people you already know via People + Face. Inbound CVs go to Growth as a conversation, not a hire.

## How do meeting notes become tasks?

Fathom records the call. The webhook files a **proposed** Delivery plan. It does not write the board.

1. Put the bet slug in the calendar title (for example `academy weekly`).
2. Join with the notetaker on. After the recording is ready, open Delivery → **Job chat**.
3. Read the plan. Click **Accept** if the tasks and people are right. Extra items stay held — the stage still caps how many can be open.

Ops Telegram `/pm` still pastes notes by hand if Fathom missed a call. Do not chase this on customer WhatsApp. `/approve` is still only the info@ draft.

## How do I mark a delivery task done?

1. Open Delivery (`/admin/delivery`).
2. Click **Done** on the task (or **Skip**).
3. Or WhatsApp `/today done …` from your own People phone. Do not use `/approve`.

## What should a field person do?

1. Read this week’s mission on the project Card (named place, named people). The this-week line is the short version.
2. Go to the place. Bring named people back.
3. You decide GO / PIVOT / KILL on the project. They often have no login. They are not a director of TheMobileProf.

## How do I track a grant deadline?

1. Office → **Grants** → Add: name, deadline, fit, next action.
2. Or `/ops` with the fund name. That writes the same row. It does not submit the application.
3. Click **Done** after the action, or **Drop**.

## How do I file a PDF into the data room?

1. On ops Telegram, send the file with **`/file`**, or drop a PDF in that chat. A caption may name the bet.
2. A catalog match files itself. Odd paths stay on Office → **Files** for Accept or Turn away.
3. Chat never writes Drive. Paste the Shared Drive id on Defaults first.

Office papers (CAC, TIN, bank letters) go under TheMobileProf, not a bet. Do not send class files to the Academy bot. Do not use `/approve`.

## How do I track a training, campus week, or dated event on a project?

1. Open the project → **Dates**.
2. Fill What, start date, Where, Who, and success. Leave “Set as this week’s line” ticked if this is the agenda. Click **Save date**.
3. Click **Add task on Delivery** for the work. After it happened, click **Delivered**.

Seat campus helpers on Face as assistant. Do not put this in Community (those are Telegram student rooms). Do not use `/approve`.

## How do I save this week’s mission?

1. Open the project → **Card**.
2. Hypothesis: what must be true this gate. Mission: named place, named people. Due if you have one.
3. Save. Field humans still go to the place. GO / PIVOT / KILL is Accept on Decide. Do not nag this on customer WhatsApp.

## How do I record spend on a bet?

1. Open the project → **Card**.
2. Paste this month’s figure in NGN from Books. Do not compute VAT here.
3. Board shows that figure on the card. Tracking categories in Books come later.
