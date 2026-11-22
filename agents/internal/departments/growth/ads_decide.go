package growth

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// A cell must see a full Lagos week (weekday + weekend) before we pause it.
const AdsCellLearnDays = 7

// Conversions need two weeks. One quiet weekday is not a kill.
const AdsConvertLearnDays = 14

const adsClickFloorForConvert = 5

var adsWeekdays = []string{"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY"}
var adsWeekendDays = []string{"SATURDAY", "SUNDAY"}
var adsAllDays = []string{"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"}

type adsWindow struct {
	ID           string
	Label        string
	Days         []string
	StartHour    int
	EndHour      int // exclusive. If EndHour < StartHour the window wraps midnight.
	Family       string
	WrapMidnight bool
}

func adsCombinedWindows() []adsWindow {
	return []adsWindow{
		{ID: "weekday_morning", Label: "weekday morning 07–11 WAT", Days: adsWeekdays, StartHour: 7, EndHour: 11, Family: "combined"},
		{ID: "weekday_midday", Label: "weekday midday 11–16 WAT", Days: adsWeekdays, StartHour: 11, EndHour: 16, Family: "combined"},
		{ID: "weekday_evening", Label: "weekday evening 16–21 WAT", Days: adsWeekdays, StartHour: 16, EndHour: 21, Family: "combined"},
		{ID: "weekend", Label: "weekend 08–21 WAT", Days: adsWeekendDays, StartHour: 8, EndHour: 21, Family: "combined"},
		{ID: "night", Label: "night 21–07 WAT", Days: adsAllDays, StartHour: 21, EndHour: 7, Family: "combined", WrapMidnight: true},
	}
}

func adsHourWindows() []adsWindow {
	return []adsWindow{
		{ID: "hour_morning", Label: "07–11 WAT", Days: adsAllDays, StartHour: 7, EndHour: 11, Family: "hour"},
		{ID: "hour_midday", Label: "11–16 WAT", Days: adsAllDays, StartHour: 11, EndHour: 16, Family: "hour"},
		{ID: "hour_evening", Label: "16–21 WAT", Days: adsAllDays, StartHour: 16, EndHour: 21, Family: "hour"},
		{ID: "hour_night", Label: "21–07 WAT", Days: adsAllDays, StartHour: 21, EndHour: 7, Family: "hour", WrapMidnight: true},
	}
}

func adsDayWindows() []adsWindow {
	return []adsWindow{
		{ID: "days_weekday", Label: "weekdays", Days: adsWeekdays, StartHour: 0, EndHour: 24, Family: "day"},
		{ID: "days_weekend", Label: "weekend", Days: adsWeekendDays, StartHour: 0, EndHour: 24, Family: "day"},
	}
}

func adsWindowByID(id string) (adsWindow, bool) {
	for _, w := range append(append(adsCombinedWindows(), adsHourWindows()...), adsDayWindows()...) {
		if w.ID == id {
			return w, true
		}
	}
	return adsWindow{}, false
}

func adsNormDay(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	switch s {
	case "MON", "MONDAY":
		return "MONDAY"
	case "TUE", "TUES", "TUESDAY":
		return "TUESDAY"
	case "WED", "WEDNESDAY":
		return "WEDNESDAY"
	case "THU", "THUR", "THURS", "THURSDAY":
		return "THURSDAY"
	case "FRI", "FRIDAY":
		return "FRIDAY"
	case "SAT", "SATURDAY":
		return "SATURDAY"
	case "SUN", "SUNDAY":
		return "SUNDAY"
	default:
		return ""
	}
}

func adsWeekend(day string) bool {
	return day == "SATURDAY" || day == "SUNDAY"
}

func adsDayFromDate(date string) string {
	date = strings.TrimSpace(date)
	if date == "" {
		return ""
	}
	if len(date) > 10 {
		date = date[:10]
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	return strings.ToUpper(t.Weekday().String())
}

func adsHourOK(hour int, hasHour bool) bool {
	return hasHour && hour >= 0 && hour <= 23
}

// AdsDaypartID maps a Lagos hour and weekday onto one cheap experiment bucket.
// Combined ids (weekday_morning…) need both; hour_* and days_* are the orthogonal Meta factors.
func AdsDaypartID(dayOfWeek string, hour int, hasHour bool) string {
	day := adsNormDay(dayOfWeek)
	if day == "" && hasHour && adsHourOK(hour, true) {
		if hour >= 7 && hour < 11 {
			return "hour_morning"
		}
		if hour >= 11 && hour < 16 {
			return "hour_midday"
		}
		if hour >= 16 && hour < 21 {
			return "hour_evening"
		}
		return "hour_night"
	}
	if day != "" && !hasHour {
		if adsWeekend(day) {
			return "days_weekend"
		}
		return "days_weekday"
	}
	if day == "" || !adsHourOK(hour, hasHour) {
		return ""
	}
	if hour >= 21 || hour < 7 {
		return "night"
	}
	if adsWeekend(day) {
		if hour >= 8 && hour < 21 {
			return "weekend"
		}
		return "night"
	}
	if hour < 11 {
		return "weekday_morning"
	}
	if hour < 16 {
		return "weekday_midday"
	}
	return "weekday_evening"
}

// AdsCellWasteReason is empty while the experiment is still learning.
func AdsCellWasteReason(ageDays, cost, clicks, conversions float64, anyConv bool) string {
	if ageDays < AdsCellLearnDays || cost <= 0 {
		return ""
	}
	if clicks < 1 {
		return "spent with 0 clicks after a week"
	}
	if ageDays < AdsConvertLearnDays || !anyConv {
		return ""
	}
	if clicks >= adsClickFloorForConvert && conversions < 1 {
		return "clicks with 0 conversions after two weeks"
	}
	return ""
}

// AdsWasteReason is the two-week conversion cutoff with tracking present.
func AdsWasteReason(cost, clicks, conversions float64) string {
	return AdsCellWasteReason(AdsConvertLearnDays, cost, clicks, conversions, true)
}

func AdsAgeDays(at string) float64 {
	at = strings.TrimSpace(at)
	if at == "" {
		return AdsConvertLearnDays
	}
	t, err := time.Parse(time.RFC3339, at)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, at)
	}
	if err != nil {
		return AdsConvertLearnDays
	}
	d := time.Since(t).Hours() / 24
	if d < 0 {
		return 0
	}
	return d
}

type AdsMetricRow struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Hour        *int    `json:"hour"`
	DayOfWeek   string  `json:"day_of_week"`
	Date        string  `json:"date"`
	Cost        float64 `json:"cost"`
	Clicks      float64 `json:"clicks"`
	Conversions float64 `json:"conversions"`
	Impressions float64 `json:"impressions"`
}

func (r AdsMetricRow) hour() (int, bool) {
	if r.Hour == nil {
		return 0, false
	}
	return *r.Hour, true
}

func (r AdsMetricRow) day() string {
	if d := adsNormDay(r.DayOfWeek); d != "" {
		return d
	}
	return adsDayFromDate(r.Date)
}

func (r AdsMetricRow) part() string {
	h, ok := r.hour()
	return AdsDaypartID(r.day(), h, ok)
}

type AdsScheduleIn struct {
	ResourceName string `json:"resource_name"`
	DayOfWeek    string `json:"day_of_week"`
	StartHour    int    `json:"start_hour"`
	EndHour      int    `json:"end_hour"`
}

type AdsDecideIn struct {
	Channel          string          `json:"channel"`
	ID               string          `json:"id"`
	CustomerID       string          `json:"customer_id"`
	LoginCustomerID  string          `json:"login_customer_id"`
	CampaignID       string          `json:"campaign_id"`
	CampaignResource string          `json:"campaign_resource"`
	AgeDays          float64         `json:"age_days"`
	Dropped          []string        `json:"dropped"`
	Rows             []AdsMetricRow  `json:"rows"`
	Schedules        []AdsScheduleIn `json:"schedules"`
}

type AdsDecideOut struct {
	Skip            bool             `json:"skip"`
	Channel         string           `json:"channel"`
	ID              string           `json:"id"`
	CustomerID      string           `json:"customer_id"`
	LoginCustomerID string           `json:"login_customer_id"`
	Note            string           `json:"note"`
	Dropped         []string         `json:"dropped"`
	GoogleBody      map[string]any   `json:"google_body,omitempty"`
	Batch           []map[string]any `json:"batch,omitempty"`
}

type adsTotals struct {
	key, name                              string
	cost, clicks, conversions, impressions float64
}

func AdsDecide(in AdsDecideIn) AdsDecideOut {
	out := AdsDecideOut{
		Skip:            true,
		Channel:         strings.ToLower(strings.TrimSpace(in.Channel)),
		ID:              strings.TrimSpace(in.ID),
		CustomerID:      strings.TrimSpace(in.CustomerID),
		LoginCustomerID: strings.TrimSpace(in.LoginCustomerID),
		Dropped:         uniqStrings(in.Dropped),
	}
	if out.LoginCustomerID == "" {
		out.LoginCustomerID = out.CustomerID
	}
	if in.AgeDays < AdsCellLearnDays {
		return out
	}
	totals := map[string]*adsTotals{}
	var anyConv bool
	var live []AdsMetricRow
	for _, r := range in.Rows {
		r.Key = strings.TrimSpace(r.Key)
		if r.Key == "" {
			continue
		}
		live = append(live, r)
		if totals[r.Key] == nil {
			totals[r.Key] = &adsTotals{key: r.Key, name: strings.TrimSpace(r.Name)}
		}
		t := totals[r.Key]
		if t.name == "" {
			t.name = strings.TrimSpace(r.Name)
		}
		t.cost += r.Cost
		t.clicks += r.Clicks
		t.conversions += r.Conversions
		t.impressions += r.Impressions
		if r.Conversions > 0 {
			anyConv = true
		}
	}
	if len(totals) == 0 {
		return out
	}
	paused := map[string]string{}
	var notes []string
	for _, t := range sortedTotals(totals) {
		reason := AdsCellWasteReason(in.AgeDays, t.cost, t.clicks, t.conversions, anyConv)
		if reason == "" {
			continue
		}
		paused[t.key] = reason
		label := t.name
		if label == "" {
			label = t.key
		}
		notes = append(notes, label+": "+reason)
	}
	var remain []AdsMetricRow
	for _, r := range live {
		if _, dead := paused[r.Key]; !dead {
			remain = append(remain, r)
		}
	}
	dropped := uniqStrings(in.Dropped)
	switch out.Channel {
	case "google":
		newDrops := wasteInFamily(remain, adsCombinedWindows(), droppedFor(dropped, ""), func(r AdsMetricRow) string { return r.part() })
		dropped = mergeDropped(dropped, "", newDrops)
		for _, id := range newDrops {
			if w, ok := adsWindowByID(id); ok {
				notes = append(notes, "cut "+w.Label)
			}
		}
		ops := googlePauseOps(paused)
		camp := strings.TrimSpace(in.CampaignResource)
		if camp == "" && out.CustomerID != "" && strings.TrimSpace(in.CampaignID) != "" {
			camp = "customers/" + out.CustomerID + "/campaigns/" + strings.TrimSpace(in.CampaignID)
		}
		if camp != "" && len(newDrops) > 0 {
			keep := servingIDs(adsCombinedWindows(), droppedFor(dropped, ""))
			ops = append(ops, googleScheduleOps(camp, in.Schedules, keep)...)
		}
		if len(ops) == 0 {
			out.Dropped = dropped
			return out
		}
		out.Skip = false
		out.Dropped = dropped
		out.Note = strings.Join(notes, "; ")
		out.GoogleBody = map[string]any{"mutateOperations": ops}
		return out
	case "meta":
		var batch []map[string]any
		keys := make([]string, 0, len(totals))
		for k := range totals {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if reason, ok := paused[key]; ok {
				batch = append(batch, map[string]any{"method": "POST", "relative_url": key, "body": "status=PAUSED"})
				_ = reason
				continue
			}
			var mine []AdsMetricRow
			for _, r := range remain {
				if r.Key == key {
					mine = append(mine, r)
				}
			}
			hourDrops := wasteInFamily(mine, adsHourWindows(), droppedFor(dropped, key), func(r AdsMetricRow) string {
				h, ok := r.hour()
				if !ok {
					return ""
				}
				return AdsDaypartID("", h, true)
			})
			dayDrops := wasteInFamily(mine, adsDayWindows(), droppedFor(dropped, key), func(r AdsMetricRow) string {
				return AdsDaypartID(r.day(), 0, false)
			})
			if len(hourDrops)+len(dayDrops) == 0 {
				continue
			}
			dropped = mergeDropped(dropped, key, append(hourDrops, dayDrops...))
			for _, id := range append(hourDrops, dayDrops...) {
				if w, ok := adsWindowByID(id); ok {
					notes = append(notes, strings.TrimSpace(totals[key].name)+": cut "+w.Label)
				}
			}
			sched := metaSchedule(servingIDs(adsHourWindows(), droppedFor(dropped, key)), servingIDs(adsDayWindows(), droppedFor(dropped, key)))
			if len(sched) == 0 {
				continue
			}
			batch = append(batch, map[string]any{
				"method":       "POST",
				"relative_url": key,
				"body":         "adset_schedule=" + url.QueryEscape(metaScheduleBody(sched)),
			})
		}
		if len(batch) == 0 {
			out.Dropped = dropped
			return out
		}
		out.Skip = false
		out.Dropped = dropped
		out.Note = strings.Join(notes, "; ")
		out.Batch = batch
		return out
	default:
		return out
	}
}

func sortedTotals(m map[string]*adsTotals) []*adsTotals {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*adsTotals, 0, len(keys))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}

type adsBucket struct {
	cost, clicks float64
}

func wasteInFamily(rows []AdsMetricRow, family []adsWindow, already []string, pick func(AdsMetricRow) string) []string {
	serving := map[string]bool{}
	for _, w := range family {
		serving[w.ID] = true
	}
	for _, id := range already {
		delete(serving, id)
	}
	if len(serving) == 0 {
		return nil
	}
	agg := map[string]*adsBucket{}
	for _, r := range rows {
		id := pick(r)
		if !serving[id] {
			continue
		}
		if agg[id] == nil {
			agg[id] = &adsBucket{}
		}
		agg[id].cost += r.Cost
		agg[id].clicks += r.Clicks
	}
	var someClick bool
	for id := range serving {
		if b := agg[id]; b != nil && b.clicks >= 1 {
			someClick = true
		}
	}
	if !someClick {
		return nil
	}
	var drops []string
	kept := 0
	for id := range serving {
		b := agg[id]
		if b != nil && b.cost > 0 && b.clicks < 1 {
			drops = append(drops, id)
			continue
		}
		kept++
	}
	if kept == 0 {
		return nil
	}
	sort.Strings(drops)
	return drops
}

func servingIDs(family []adsWindow, dropped []string) map[string]bool {
	dead := map[string]bool{}
	for _, id := range dropped {
		dead[id] = true
	}
	keep := map[string]bool{}
	for _, w := range family {
		if !dead[w.ID] {
			keep[w.ID] = true
		}
	}
	return keep
}

func droppedFor(dropped []string, key string) []string {
	var out []string
	prefix := key + ":"
	for _, d := range dropped {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if key == "" {
			if !strings.Contains(d, ":") {
				out = append(out, d)
			}
			continue
		}
		if strings.HasPrefix(d, prefix) {
			out = append(out, strings.TrimPrefix(d, prefix))
		}
	}
	return uniqStrings(out)
}

func mergeDropped(dropped []string, key string, extra []string) []string {
	out := append([]string{}, dropped...)
	for _, id := range extra {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if key != "" {
			id = key + ":" + id
		}
		out = append(out, id)
	}
	return uniqStrings(out)
}

func uniqStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func googlePauseOps(paused map[string]string) []map[string]any {
	keys := make([]string, 0, len(paused))
	for k := range paused {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var ops []map[string]any
	for _, k := range keys {
		ops = append(ops, map[string]any{
			"adGroupOperation": map[string]any{
				"update":     map[string]any{"resourceName": k, "status": "PAUSED"},
				"updateMask": "status",
			},
		})
	}
	return ops
}

func googleScheduleOps(campaign string, existing []AdsScheduleIn, keep map[string]bool) []map[string]any {
	var ops []map[string]any
	seen := map[string]bool{}
	for _, s := range existing {
		rn := strings.TrimSpace(s.ResourceName)
		if rn == "" || seen[rn] {
			continue
		}
		seen[rn] = true
		ops = append(ops, map[string]any{"campaignCriterionOperation": map[string]any{"remove": rn}})
	}
	for _, slot := range adsSlotsFromKeep(keep) {
		ops = append(ops, map[string]any{
			"campaignCriterionOperation": map[string]any{
				"create": map[string]any{
					"campaign": campaign,
					"adSchedule": map[string]any{
						"dayOfWeek":   slot.DayOfWeek,
						"startHour":   slot.StartHour,
						"startMinute": "ZERO",
						"endHour":     slot.EndHour,
						"endMinute":   "ZERO",
					},
				},
			},
		})
	}
	return ops
}

type adsSlot struct {
	DayOfWeek string
	StartHour int
	EndHour   int
}

func adsSlotsFromKeep(keep map[string]bool) []adsSlot {
	var slots []adsSlot
	for _, w := range adsCombinedWindows() {
		if !keep[w.ID] {
			continue
		}
		for _, day := range w.Days {
			if w.WrapMidnight {
				slots = append(slots, adsSlot{DayOfWeek: day, StartHour: w.StartHour, EndHour: 24})
				slots = append(slots, adsSlot{DayOfWeek: day, StartHour: 0, EndHour: w.EndHour})
				continue
			}
			slots = append(slots, adsSlot{DayOfWeek: day, StartHour: w.StartHour, EndHour: w.EndHour})
		}
	}
	sort.Slice(slots, func(i, j int) bool {
		if slots[i].DayOfWeek != slots[j].DayOfWeek {
			return dayIndex(slots[i].DayOfWeek) < dayIndex(slots[j].DayOfWeek)
		}
		return slots[i].StartHour < slots[j].StartHour
	})
	return slots
}

func dayIndex(d string) int {
	for i, x := range adsAllDays {
		if x == d {
			return i
		}
	}
	return 99
}

func metaSchedule(hours, days map[string]bool) []map[string]any {
	dayNums := metaDays(days)
	if len(dayNums) == 0 {
		return nil
	}
	ranges := metaHourRanges(hours)
	if len(ranges) == 0 {
		return nil
	}
	if len(hours) == len(adsHourWindows()) && len(days) == len(adsDayWindows()) {
		return nil
	}
	var out []map[string]any
	for _, r := range ranges {
		out = append(out, map[string]any{
			"start_minute":  r[0],
			"end_minute":    r[1],
			"days":          dayNums,
			"timezone_type": "USER",
		})
	}
	return out
}

func metaDays(days map[string]bool) []int {
	want := map[string]bool{}
	if days["days_weekday"] {
		for _, d := range adsWeekdays {
			want[d] = true
		}
	}
	if days["days_weekend"] {
		for _, d := range adsWeekendDays {
			want[d] = true
		}
	}
	// Meta: 0=Sunday … 6=Saturday
	order := []string{"SUNDAY", "MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY"}
	var out []int
	for i, d := range order {
		if want[d] {
			out = append(out, i)
		}
	}
	return out
}

func metaHourRanges(hours map[string]bool) [][2]int {
	var ranges [][2]int
	for _, w := range adsHourWindows() {
		if !hours[w.ID] {
			continue
		}
		if w.WrapMidnight {
			ranges = append(ranges, [2]int{w.StartHour * 60, 24 * 60})
			ranges = append(ranges, [2]int{0, w.EndHour * 60})
			continue
		}
		ranges = append(ranges, [2]int{w.StartHour * 60, w.EndHour * 60})
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i][0] < ranges[j][0] })
	return ranges
}

func metaScheduleBody(sched []map[string]any) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, s := range sched {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"start_minute":`)
		b.WriteString(strconv.Itoa(intVal(s["start_minute"])))
		b.WriteString(`,"end_minute":`)
		b.WriteString(strconv.Itoa(intVal(s["end_minute"])))
		b.WriteString(`,"days":[`)
		days, _ := s["days"].([]int)
		for j, d := range days {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Itoa(d))
		}
		b.WriteString(`],"timezone_type":"USER"}`)
	}
	b.WriteByte(']')
	return b.String()
}
