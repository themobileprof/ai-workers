# Onsite SEO is Cursor in the bet repo

There is no SEO worker, no `/seo` desk, and no pack URL. Open the public site’s GitHub repo in Cursor. Copy `.cursor/rules/onsite-seo.mdc` from this office into that repo if it is missing. Walk the **source**, not fetched HTML.

Legal name: TheMobileProf Technologies. Bets (Academy, HomeGauge, Mechazone, MomLaunchpad, Finchest) are pages under that company unless the page already names another legal entity.

## Source of truth

The visible page (H1, headings, body, buttons) is truth. Title tags and meta are chrome — a designer may never have seen them.

- **gap** — said on the designed page, missing from the title tag.
- **aligned** — visible copy and title agree.
- **brochure** — title/meta only; do not treat it as the topic.
- **cannibal** — the same visible phrase is strong on two URLs.

Rewrite visible copy into how a paying person in Lagos / West Africa would type. Every content word in a proposed query must map to an entity already on the page or the Card one-liner. If Ikeja is not on the page, do not add Ikeja. No competitor domains. No invented volume, rankings, Core Web Vitals, or ChatGPT citations.

## Walk (one step at a time)

1. **Cite** — 3–6 factual sentences already on the page (who, where, offer, constraint, hours, price if shown). No invented stats.
2. **FAQ** — questions a Lagos searcher would type that the page already answers. Visible Q&A first, then FAQPage JSON-LD that matches.
3. **Schema** — JSON-LD only for types the designed page supports. Organization is TheMobileProf Technologies unless the page names someone else. LocalBusiness / Course / Offer / FAQPage only from attested facts. Flag if GPTBot or Google-Extended is disallowed.
4. **Onsite** — weave cite + FAQ + schema into the page. Visible copy first (H1, subheads, body, CTAs, same voice and offers). Chrome second: title ≤60 characters, meta ≤155, matching the designed H1/body.
5. **Offsite** — listings and partners this company already knows. Add ordinary Delivery tasks, spread over weeks. No cold email, WhatsApp blast, or bought links.

One primary phrase per URL. Cluster synonyms; one URL owns the cluster. Keep negatives from the copy (not the lender, invitation-only, no public checkout). Drop nav junk.

## Measurement (office, not Cursor)

Search Console and GA4 property ids live on Defaults (`/admin/settings` → Traffic). n8n pulls last-28-days as `ga4-read`. Quote only those snapshots. Cursor in the bet repo does not call Google APIs.

## How you start

1. Put the public URL (and GitHub repo) on the project Card.
2. Open that repo in Cursor. Paste or copy `.cursor/rules/onsite-seo.mdc`.
3. Do the walk in source. Humans review the diff. This office does not push PRs.
4. Offsite listings you already have: Delivery → Add a task.
