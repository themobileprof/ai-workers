package admin

// SettingField is one company-desk default. Accounts/n8n read these via
// GET /internal/v1/settings. Secrets (Paystack, LLM, Zoho OAuth) do not live here.
type SettingField struct {
	Key   string
	Label string
	Help  string
}

type settingGroupSpec struct {
	Title  string
	Help   string
	Fields []SettingField
}

func settingCatalog() []settingGroupSpec {
	return []settingGroupSpec{
		{
			Title: "Company",
			Help:  "Shown on the desk and used as the Lagos clock for Books dates.",
			Fields: []SettingField{
				{Key: "company.name", Label: "Legal name", Help: "TheMobileProf Technologies in Books."},
				{Key: "company.timezone", Label: "Timezone", Help: "IANA name. Expense and invoice dates use this when the chat has no date."},
			},
		},
		{
			Title: "Zoho Books · accounts",
			Help:  "n8n Books write and Paystack paid read these on every run. Copy the ids from Books → Accountant → Chart of Accounts (hover the account). Go never computes tax.",
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
			Title: "Paystack · public",
			Help:  "The secret key stays in the VM .env (n8n only). This row is currency for Payment Requests.",
			Fields: []SettingField{
				{Key: "paystack.currency", Label: "Charge currency", Help: "NGN. Amount in kobo is Books invoice.total × 100."},
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
	Title  string
	Help   string
	Fields []SettingFieldView
}

func settingGroupsFrom(values map[string]string) []SettingGroupView {
	seen := catalogSettingKeys()
	var groups []SettingGroupView
	for _, g := range settingCatalog() {
		view := SettingGroupView{Title: g.Title, Help: g.Help}
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
		groups = append(groups, SettingGroupView{Title: "Other", Help: "Keys on this desk that are not in the accounts catalog.", Fields: extra})
	}
	return groups
}
