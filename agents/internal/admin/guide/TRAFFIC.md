# Traffic — {Bet name}

Legal name: **TheMobileProf Technologies**. Campaign name (ads) is the slug `{slug}`.

This file is the brief for Cursor while building pages, events, and ads landings. Update it in the **same diff** as a new public URL or a new GA4 / `gtag` event. Do not list an event the code does not fire.

Copy this file to the public site repo root as `TRAFFIC.md`. Copy `.cursor/rules/traffic.mdc` and `.cursor/rules/onsite-seo.mdc` from the office into that repo. Set `alwaysApply: true` on `traffic.mdc` there.

## Offer

- One-liner (same as the project Card):
- Who pays:
- Public URL:
- Primary CTA (button text already on the page):

## Search (one phrase per URL)

| URL | Primary phrase | Role | Notes |
| --- | --- | --- | --- |
| `/` | | gap / aligned / brochure | From the designed H1/body only |

Negatives (from the copy, not invented):

## Funnel (GA4 events that already fire)

Paste this line onto the project Card (`funnel_steps`). If this bet is the one spending, paste it on Defaults → Ads as well.

```
Session:session_start, Landing:page_view
```

A third step only when that event exists, for example `Lead:generate_lead` or a path `Waitlist:/waitlist`.

## Ads landing

- Final URL = Card public URL. Place stamps `utm_campaign` to `{slug}` and `utm_content` to the geo/theme cell.
- Geo and keyword tests use words already in Offer or on the page. Not age or gender.

## Do not

- Invent volume, rankings, cities, or a checkout the site does not have.
- Treat this file as live GA4 numbers. Those snapshots live on the office desk.
- Publish, merge, or spend. Humans review the diff.
