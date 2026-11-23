package admin

// SettingField is one company-desk default. Accounts/n8n read these via
// GET /internal/v1/settings. Secrets (Paystack, LLM, Zoho OAuth) do not live here.
type SettingField struct {
	Key   string
	Label string
	Help  string
}

type settingGroupSpec struct {
	ID       string
	Title    string
	Help     string
	Recipe   []string
	DocHref  string
	DocLabel string
	Fields   []SettingField
}

func settingCatalog() []settingGroupSpec {
	return []settingGroupSpec{
		{
			ID:       "company",
			Title:    "Company",
			Help:     "Shown on the desk and used as the Lagos clock for Books dates.",
			DocHref:  "/admin/docs/desk",
			DocLabel: "Company desk sheet",
			Recipe: []string{
				"Legal name matches CAC / Books — TheMobileProf Technologies.",
				"Timezone is Africa/Lagos unless you have a reason to change it.",
				"Click Save on this tab only. Other vendors keep their values.",
			},
			Fields: []SettingField{
				{Key: "company.name", Label: "Legal name", Help: "TheMobileProf Technologies in Books."},
				{Key: "company.timezone", Label: "Timezone", Help: "IANA name. Expense and invoice dates use this when the chat has no date."},
			},
		},
		{
			ID:       "zoho",
			Title:    "Zoho Books · accounts",
			Help:     "n8n Books write and Paystack paid read these on every run. Copy the ids from Books → Accountant → Chart of Accounts (hover the account). Go never computes tax.",
			DocHref:  "/admin/docs/accounts",
			DocLabel: "Accounts sheet",
			Recipe: []string{
				"Open Books → Accountant → Chart of Accounts.",
				"Hover the account, copy the id. Do not paste API keys or OAuth tokens.",
				"Click Save on this tab. Paystack and Drive stay as they are.",
			},
			Fields: []SettingField{
				{Key: "zoho.organization_id", Label: "Organization id", Help: "Books org TheMobileProf Technologies."},
				{Key: "zoho.paid_through_account_id", Label: "Paid-through account", Help: "Petty Cash (or the cash/bank you spend from) for already-paid expenses."},
				{Key: "zoho.default_expense_account_id", Label: "Default expense account", Help: "Other Expenses — used when the classifier does not pick a tighter category."},
				{Key: "zoho.account.office_supplies_id", Label: "Office Supplies", Help: "Maps classifier zoho_account_name Office Supplies."},
				{Key: "zoho.account.advertising_id", Label: "Advertising And Marketing", Help: "Maps advertising / ads / boost."},
				{Key: "zoho.account.lodging_id", Label: "Lodging", Help: "Maps lodging / hotel / Airbnb."},
				{Key: "zoho.account.uncategorized_id", Label: "Uncategorized", Help: "Maps uncategorized spend."},
				{Key: "zoho.deposit_to_account_id", Label: "Paystack deposit account", Help: "A Bank account in Books for customerpayments. Leave blank to use Zoho’s default deposit account. Not Petty Cash."},
			},
		},
		{
			ID:       "paystack",
			Title:    "Paystack · public",
			Help:     "The secret key stays in the VM .env (n8n only). This row is currency for Payment Requests.",
			DocHref:  "/admin/docs/accounts",
			DocLabel: "Accounts sheet",
			Recipe: []string{
				"Currency is NGN unless a Payment Request is in another currency.",
				"The secret key stays in the VM .env — never paste it here.",
				"Click Save on this tab only.",
			},
			Fields: []SettingField{
				{Key: "paystack.currency", Label: "Charge currency", Help: "NGN. Amount in kobo is Books invoice.total × 100."},
			},
		},
		{
			ID:       "drive",
			Title:    "Google Drive · data room",
			Help:     "Shared Drive id (not a My Drive folder). OAuth stays in n8n. Accept on Files writes here; the worker never uploads.",
			DocHref:  "/admin/docs/files",
			DocLabel: "Files sheet",
			Recipe: []string{
				"Open the Shared Drive in Google Drive (not a personal My Drive folder).",
				"Copy the id from the URL after /folders/.",
				"Click Save on this tab. The n8n credential must be named Google Drive.",
			},
			Fields: []SettingField{
				{Key: "drive.shared_drive_id", Label: "Shared Drive id", Help: "From Drive URL /drive/u/0/folders/… on a Shared Drive. Leave blank until the n8n Google Drive credential exists."},
			},
		},
		{
			ID:       "community",
			Title:    "Academy · join",
			Help:     "Student invite links are https://lms.themobileprof.com/join/CODE. ClubKonnect secrets stay in the VM .env. Telegram invite is the group link students receive after they say hi to the host.",
			DocHref:  "/admin/docs/community",
			DocLabel: "Community sheet",
			Recipe: []string{
				"LMS register is the student signup URL. Default is the LMS /join page.",
				"Telegram invite is t.me/+… for the class group. Leave blank and the host will try to create one when a student /start’s.",
				"Bot username is without @. Airtime amount is REWARD_AMOUNT in .env.",
			},
			Fields: []SettingField{
				{Key: "community.lms_register", Label: "LMS register URL", Help: "Where a new student creates an Academy account."},
				{Key: "community.telegram_invite", Label: "Telegram group invite", Help: "t.me/+… link. Students get this after they open the host in Telegram."},
				{Key: "community.telegram_bot", Label: "Academy bot username", Help: "Without @. Used for t.me/bot?start=r_CODE."},
			},
		},
		{
			ID:       "seo",
			Title:    "Traffic · Google properties",
			Help:     "Property ids only — free Google APIs for THIS company. The GA4 JSON key stays in n8n. PageSpeed key stays in the VM .env. Onsite SEO is Cursor in the bet repo.",
			DocHref:  "/admin/docs/seo",
			DocLabel: "SEO sheet",
			Recipe: []string{
				"Search Console property is the verified site (https://lms.themobileprof.com/ or sc-domain:themobileprof.com). Add ga4-read@themobileprof.iam.gserviceaccount.com as a user on that property.",
				"GA4 property id is the digits from Admin → Property settings. Add the same email as Viewer on that property.",
				"Click Save on this tab. n8n credential Google Analytics (service account) pulls GA4 and Search Console. Add webmasters.readonly to its HTTP scopes. PAGESPEED_API_KEY stays in .env.",
			},
			Fields: []SettingField{
				{Key: "seo.gsc_site_url", Label: "Search Console property", Help: "Your verified site URL or sc-domain:…. n8n reads this as ga4-read. Queries, clicks, impressions, position."},
				{Key: "seo.ga4_property_id", Label: "GA4 property id", Help: "Digits only. n8n reads this. The ga4-read service account must be a Viewer on that property. Traffic only, not rankings."},
			},
		},
		{
			ID:       "ads",
			Title:    "Ads · Google and Meta",
			Help:     "Account ids only. OAuth stays in n8n. Google sunset developer tokens on 9 September 2026; API access is the Cloud project that owns the Google Ads credential. Go never holds Ads tokens. After Accept, Book it yourself or Place — not both for the same channel.",
			DocHref:  "/admin/docs/growth",
			DocLabel: "Growth sheet",
			Recipe: []string{
				"Google Ads customer id is the 10 digits (hyphens optional). Login customer id is the MCC if you use a manager account.",
				"Meta ad account is act_… or the digits. Page id is the Facebook Page that will sponsor the ad.",
				"Place is conservative for a self-funded shop: platform-minimum daily probe, never more than remaining/90, exact match on Search, Google Search only (no partner network). Cells are separate experiments. Place runs all hours for a week. A 07:00 Lagos job then pauses a cell that spent with 0 clicks, and turns off Lagos dayparts that wasted while another daypart got clicks. Conversions wait two weeks, and only if a conversion already exists. Daily cap is an optional extra ceiling. Required if the account has no remaining amount (open credit line).",
				"Campaigns start PAUSED unless you set ENABLED. Click Save. n8n credentials Google Ads and Facebook Ads. Enable the Google Ads API on the Cloud project that owns the Google Ads OAuth client, then apply for Explorer (or higher) so production accounts work.",
				"GA4 (same property id on Traffic) scores Place: last 7 days of paid sessions, and a funnel of events that already fire. Prefer the Funnel line on the bet Card (TRAFFIC.md in the site repo). This field is the company-wide fallback. Default Session → Landing. Add a lead event only if that event exists on the site. Link Google Ads to GA4 for auto-tagging. Place also stamps utm_campaign to the bet slug. After the morning pull, a proposed glance job on Growth quotes the snapshot. It does not Place.",
			},
			Fields: []SettingField{
				{Key: "ads.google_customer_id", Label: "Google Ads customer id", Help: "10 digits. From ads.google.com. Not an OAuth token."},
				{Key: "ads.google_login_customer_id", Label: "Google Ads login (MCC) id", Help: "Manager account id if you access the customer through MCC. Leave blank if you log in as the customer."},
				{Key: "ads.meta_ad_account_id", Label: "Meta ad account id", Help: "Digits, with or without act_. Ads Manager → account settings."},
				{Key: "ads.meta_page_id", Label: "Facebook Page id", Help: "The Page that sponsors the ad. Required for Meta Place."},
				{Key: "ads.daily_cap", Label: "Daily ceiling (optional)", Help: "Max daily budget in the ads account currency. Place already uses the platform minimum. This only clips further. Required if the account does not report remaining funds."},
				{Key: "ads.campaign_status", Label: "Campaign status", Help: "PAUSED (default) or ENABLED. Place uses this. ENABLED starts spend."},
				{Key: "ads.funnel_steps", Label: "Funnel steps", Help: "Company-wide fallback. Prefer the Funnel line on each bet Card (TRAFFIC.md). Comma-separated Label:event. Path: Label:/waitlist. Need two or more. Blank is Session:session_start, Landing:page_view. A third step only if that event already fires."},
			},
		},
	}
}

func defaultSettings() map[string]any {
	return map[string]any{
		"company.name":                    "TheMobileProf Technologies",
		"company.timezone":                "Africa/Lagos",
		"zoho.organization_id":            "939049468",
		"zoho.paid_through_account_id":    "1300646000000000361",
		"zoho.default_expense_account_id": "1300646000000000460",
		"zoho.account.office_supplies_id": "1300646000000000400",
		"zoho.account.advertising_id":     "1300646000000000403",
		"zoho.account.lodging_id":         "1300646000000032023",
		"zoho.account.uncategorized_id":   "1300646000000035005",
		"zoho.deposit_to_account_id":      "",
		"paystack.currency":               "NGN",
		"drive.shared_drive_id":           "",
		"seo.gsc_site_url":                "",
		"seo.ga4_property_id":             "",
		"community.lms_register":          "https://lms.themobileprof.com/join",
		"community.telegram_invite":       "https://t.me/+6lTlfZkJ0TU3ZGE0",
		"community.telegram_bot":          "themobileprof_academy_bot",
		"ads.google_customer_id":          "",
		"ads.google_login_customer_id":    "",
		"ads.meta_ad_account_id":          "",
		"ads.meta_page_id":                "",
		"ads.daily_cap":                   "",
		"ads.campaign_status":             "PAUSED",
		"ads.funnel_steps":                "",
	}
}

func catalogSettingKeys() map[string]bool {
	out := map[string]bool{}
	for _, g := range settingCatalog() {
		for _, f := range g.Fields {
			out[f.Key] = true
		}
	}
	return out
}

type SettingFieldView struct {
	SettingField
	Value string
}

type SettingGroupView struct {
	ID       string
	Title    string
	Help     string
	Recipe   []string
	DocHref  string
	DocLabel string
	Fields   []SettingFieldView
}

func settingGroupsFrom(values map[string]string) []SettingGroupView {
	seen := catalogSettingKeys()
	var groups []SettingGroupView
	for _, g := range settingCatalog() {
		view := SettingGroupView{ID: g.ID, Title: g.Title, Help: g.Help, Recipe: g.Recipe, DocHref: g.DocHref, DocLabel: g.DocLabel}
		for _, f := range g.Fields {
			view.Fields = append(view.Fields, SettingFieldView{SettingField: f, Value: values[f.Key]})
		}
		groups = append(groups, view)
	}
	var extra []SettingFieldView
	for _, row := range toKV(values) {
		if seen[row.Key] {
			continue
		}
		extra = append(extra, SettingFieldView{
			SettingField: SettingField{Key: row.Key, Label: row.Key},
			Value:        row.Value,
		})
	}
	if len(extra) > 0 {
		groups = append(groups, SettingGroupView{
			ID:     "other",
			Title:  "Other",
			Help:   "Keys on this desk that are not in the accounts catalog.",
			Recipe: []string{"Change a key only if you know why it is here.", "Click Save on this tab. Catalog tabs keep their values."},
			Fields: extra,
		})
	}
	return groups
}
