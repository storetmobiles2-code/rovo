package seed

// The demo dataset. Everything here is fictional: restaurant names, people and phone numbers are
// invented, phones sit in the reserved fake range +9199999000xx (doc 10 §15.2), and no real
// business or brand is referenced. Geography marked [ASSUMPTION] must be verified by local ops
// before any production use; the seed itself is refused when ROVO_ENV=prod.

// City constants (doc 10 §15.1). Centroid per the public coordinate 16°44′56″N 78°00′13″E.
const (
	CityCode     = "MBNR"
	CitySlug     = "mahabubnagar"
	CityName     = "Mahabubnagar"
	CityNameTe   = "మహబూబ్‌నగర్" // [ASSUMPTION] spelling to be verified by a native speaker
	CityState    = "Telangana"
	CityGSTState = "36"
	CityTimezone = "Asia/Kolkata"
	CityLat      = 16.7488
	CityLng      = 78.0035
	DefaultPIN   = "509001"

	ZoneCode = "MBNR-CORE"
	ZoneName = "Mahabubnagar Core"

	AdminEmail = "admin@rovo.local"
	AdminName  = "rovo Super Admin"
)

// ZoneGeoJSON is the fixture octagon of ~5 km radius around the city centroid (doc 16 §10.1).
// [ASSUMPTION — synthetic fixture, not the real service area; the real polygon is drawn by ops.]
const ZoneGeoJSON = `{"type":"Polygon","coordinates":[[
[78.0505,16.7488],[78.0367,16.7806],[78.0035,16.7938],[77.9703,16.7806],
[77.9565,16.7488],[77.9703,16.7170],[78.0035,16.7038],[78.0367,16.7170],
[78.0505,16.7488]]]}`

type locality struct {
	Slug, Name, NameTe string
	Lat, Lng           float64
	Aliases            []string
}

// localities: names come from the candidate list in doc 10 §15.1; every centroid is an
// [ASSUMPTION – verify with local ops] placed inside the fixture zone, not a surveyed point.
var localities = []locality{
	{"new-town", "New Town", "న్యూ టౌన్", 16.7460, 78.0050, []string{"newtown"}},
	{"old-town", "Old Town", "ఓల్డ్ టౌన్", 16.7420, 77.9950, []string{"patha town"}},
	{"padmavathi-colony", "Padmavathi Colony", "పద్మావతి కాలనీ", 16.7550, 78.0150, []string{"padmavati colony"}},
	{"christianpally", "Christianpally", "క్రిస్టియన్‌పల్లి", 16.7560, 77.9850, []string{"christian colony", "christianpalli"}},
	{"rajendra-nagar", "Rajendra Nagar", "రాజేంద్ర నగర్", 16.7380, 78.0100, []string{"rajendranagar"}},
	{"mettugadda", "Mettugadda", "మెట్టుగడ్డ", 16.7550, 78.0000, []string{"metugadda"}},
	{"yenugonda", "Yenugonda", "యేనుగొండ", 16.7400, 78.0300, []string{"enugonda"}},
	{"boyapalle", "Boyapalle", "బోయపల్లి", 16.7350, 77.9800, []string{"boyapalli", "boyapally"}},
	{"bhagiratha-colony", "Bhagiratha Colony", "భగీరథ కాలనీ", 16.7480, 78.0200, []string{"bhagirath colony"}},
	{"bus-stand-area", "Bus Stand Area", "బస్ స్టాండ్ ఏరియా", 16.7450, 78.0000, []string{"bus stand", "rtc bus stand"}},
	{"clock-tower-area", "Clock Tower Area", "క్లాక్ టవర్ ఏరియా", 16.7470, 77.9990, []string{"clock tower"}},
	{"teachers-colony", "Teachers Colony", "టీచర్స్ కాలనీ", 16.7600, 78.0080, []string{"teacher colony"}},
}

type slab struct {
	FromM    int   `json:"fromM"`
	ToM      int   `json:"toM"`
	FeePaise int64 `json:"feePaise"`
}

// feeSlabs are the R18 delivery-fee slabs on road-adjusted metres, [from, to).
var feeSlabs = []slab{
	{0, 2000, 2000}, {2000, 4000, 3000}, {4000, 6000, 4000}, {6000, 8000, 5000}, {8000, 10000, 6000},
}

// Commercial defaults (doc 16 §6.5). PlatformFeePaise is 0 per R59: there is no platform fee.
const (
	PlatformFeePaise        = 0
	SmallCartThresholdPaise = 14900
	SmallCartFeePaise       = 1500
	RoadFactorMilli         = 1300
	MaxServiceableRadiusM   = 7000
	CODMaxOrderPaise        = 100000 // R6
	CODFirstOrderMaxPaise   = 60000  // R6
	RiderBasePayPaise       = 2500
	RiderBaseDistanceM      = 2000
	RiderPerKmPaise         = 600
	RiderCashLimitPaise     = 200000 // R6
	DefaultCommissionBps    = 1500
)

// etaParams are the [ASSUMPTION] two-wheeler defaults of doc 16 §5.
const etaParams = `{"pickupBufferMin":5,"handoverMin":2,"speedKmphByBand":[
{"from":"06:00","to":"11:00","kmph":22},{"from":"11:00","to":"18:00","kmph":20},
{"from":"18:00","to":"22:00","kmph":16},{"from":"22:00","to":"06:00","kmph":24}]}`

// cityConfig are the CITY-scoped operational app_config values (doc 10 §15.3). The GLOBAL timer
// and threshold defaults are reference data owned by migration 00011.
var cityConfig = []struct{ Key, JSON string }{
	{"payouts.rider.schedule", `{"frequency":"WEEKLY","runDay":"MON","period":"PREV_MON_SUN"}`},
	{"payouts.restaurant.schedule", `{"frequency":"WEEKLY","runDay":"TUE","period":"PREV_MON_SUN"}`},
	{"payouts.missed_run_alert_local_time", `"12:00"`},
	{"cod.cash_ageing_alert_h", `24`},
	{"cod.cash_ageing_block_h", `48`},
	{"cod.service_window", `"SERVICE_HOURS"`},
	{"support.phone_e164", `"+919999900000"`}, // fake demo number
	{"city.service_hours", `{"opensAt":"07:00","closesAt":"23:00"}`},
}

// reasonLabelsTe adds Telugu labels to the customer- and restaurant-facing reason codes that
// migration 00011 seeds in English. [ASSUMPTION] translations to be reviewed by a native speaker.
var reasonLabelsTe = []struct{ Category, Code, Te string }{
	{"ORDER_CANCEL", "CUSTOMER_CHANGED_MIND", "నా మనసు మార్చుకున్నాను"},
	{"ORDER_CANCEL", "CUSTOMER_ORDERED_BY_MISTAKE", "పొరపాటున ఆర్డర్ చేశాను"},
	{"ORDER_CANCEL", "CUSTOMER_WRONG_ADDRESS", "డెలివరీ చిరునామా తప్పు"},
	{"ORDER_CANCEL", "CUSTOMER_TAKING_TOO_LONG", "చాలా ఆలస్యం అవుతోంది"},
	{"ORDER_CANCEL", "CUSTOMER_OTHER", "ఇతర కారణం"},
	{"ORDER_REJECT", "ITEMS_OUT_OF_STOCK", "ఐటమ్స్ స్టాక్‌లో లేవు"},
	{"ORDER_REJECT", "TOO_BUSY", "వంటగది చాలా బిజీగా ఉంది"},
	{"ORDER_REJECT", "CLOSING_SOON", "త్వరలో మూసివేస్తున్నాం"},
	{"ORDER_REJECT", "KITCHEN_ISSUE", "వంటగది సమస్య"},
	{"ORDER_REJECT", "RESTAURANT_OTHER", "ఇతర కారణం"},
	{"DELIVERY_FAIL", "CUSTOMER_UNREACHABLE", "కస్టమర్ అందుబాటులో లేరు"},
	{"DELIVERY_FAIL", "CUSTOMER_REFUSED", "కస్టమర్ ఆర్డర్ తిరస్కరించారు"},
	{"DELIVERY_FAIL", "WRONG_OR_INCOMPLETE_ADDRESS", "చిరునామా తప్పు లేదా అసంపూర్ణం"},
}

// ledgerAccount is one platform-owned account of the doc 14 §10.2 chart.
type ledgerAccount struct {
	Code, Type, Side string
}

var platformAccounts = []ledgerAccount{
	{"PA_CLEARING", "ASSET", "D"},
	{"BANK", "ASSET", "D"},
	{"CUSTOMER_ADVANCES", "LIABILITY", "C"},
	{"REFUNDS_PAYABLE", "LIABILITY", "C"},
	{"GST_OUTPUT_9_5_RESTAURANT", "LIABILITY", "C"},
	{"GST_OUTPUT_9_5_DELIVERY", "LIABILITY", "C"},
	{"GST_OUTPUT_OWN", "LIABILITY", "C"},
	{"GST_INPUT_CREDIT", "ASSET", "D"},
	{"TDS_194O_PAYABLE", "LIABILITY", "C"},
	{"REVENUE_COMMISSION", "REVENUE", "C"},
	{"REVENUE_DELIVERY_FEE", "REVENUE", "C"},
	{"REVENUE_PLATFORM_FEE", "REVENUE", "C"},
	{"EXPENSE_RIDER_PAY", "EXPENSE", "D"},
	{"EXPENSE_PA_FEES", "EXPENSE", "D"},
	{"EXPENSE_PROMOTIONS", "EXPENSE", "D"},
	{"EXPENSE_GOODWILL", "EXPENSE", "D"},
	{"ROUND_OFF", "REVENUE", "C"},
	{"SUSPENSE", "ASSET", "D"}, // [ASSUMPTION] doc 14 leaves the type open; must trend to zero
}

type coupon struct {
	Code, Title, TitleTe, Terms string
	Type                        string // FLAT | PERCENT
	FlatPaise                   int64
	PercentBps                  int
	MaxDiscountPaise            int64
	MinOrderPaise               int64
	PerUserLimit                int
	NewUsersOnly                bool
}

var coupons = []coupon{
	{Code: "WELCOME50", Title: "50% off your first order", TitleTe: "మీ మొదటి ఆర్డర్‌పై 50% తగ్గింపు",
		Terms: "50% off the item total up to ₹75 on your first order of ₹149 or more. One use per customer.",
		Type:  "PERCENT", PercentBps: 5000, MaxDiscountPaise: 7500, MinOrderPaise: 14900, PerUserLimit: 1, NewUsersOnly: true},
	{Code: "FLAT30", Title: "₹30 off", TitleTe: "₹30 తగ్గింపు",
		Terms: "₹30 off the item total on orders of ₹199 or more. Up to 3 uses per customer.",
		Type:  "FLAT", FlatPaise: 3000, MinOrderPaise: 19900, PerUserLimit: 3},
}

type rider struct {
	Phone, Name, Vehicle, RegNo string
}

var riders = []rider{
	{"+919999900021", "Demo Rider Ravi", "MOTORCYCLE", "TS06AB1001"},
	{"+919999900022", "Demo Rider Saleem", "SCOOTER", "TS06AC1002"},
	{"+919999900023", "Demo Rider Lakshmi", "EV_SCOOTER", "TS06AD1003"},
	{"+919999900024", "Demo Rider Yadagiri", "BICYCLE", ""},
}

// Veg types.
const (
	veg    = "VEG"
	nonVeg = "NON_VEG"
	egg    = "EGG"
)

type variant struct {
	Name, NameTe string
	Rupees       int64
	Default      bool
}

type addon struct {
	Name, NameTe string
	Rupees       int64
	Veg          string
}

type addonGroup struct {
	Key, Name, NameTe string
	Min, Max          int
	Addons            []addon
}

type item struct {
	Name, NameTe, Desc string
	Veg                string
	Rupees             int64 // base price; for items with variants, the default variant's price
	PackagingRupees    int64
	Variants           []variant
	AddonGroups        []string // addonGroup keys
	Recommended        bool
	Spice              int
}

type category struct {
	Name, NameTe string
	Items        []item
}

type restaurant struct {
	Slug, Name, NameTe, Desc string
	Diet                     string // PURE_VEG | MIXED
	Lat, Lng                 float64
	Locality                 string
	Address, Landmark        string
	Phone                    string
	OwnerPhone, OwnerName    string
	Cuisines                 []string // first is primary
	PrepMin                  int
	CostForTwoRupees         int64
	RadiusM                  int
	Opens, Closes            string
	AddonGroups              []addonGroup
	Categories               []category
}

var halfFull = func(half, full int64) []variant {
	return []variant{{"Half", "హాఫ్", half, false}, {"Full", "ఫుల్", full, true}}
}

var restaurants = []restaurant{
	{
		Slug: "demo-biryani-house", Name: "Demo Biryani House", NameTe: "డెమో బిర్యానీ హౌస్",
		Desc: "Dum biryani, kebabs and curries. Fictional demo outlet.", Diet: "MIXED",
		Lat: 16.7488, Lng: 78.0035, Locality: "new-town",
		Address: "Shop 1, Demo Complex, New Town", Landmark: "Near the demo clock tower",
		Phone: "+919999900001", OwnerPhone: "+919999900011", OwnerName: "Demo Owner Imran",
		Cuisines: []string{"BIRYANI", "KEBABS_GRILL", "NORTH_INDIAN"}, PrepMin: 25, CostForTwoRupees: 400, RadiusM: 7000,
		Opens: "11:00", Closes: "23:00",
		AddonGroups: []addonGroup{
			{Key: "extras", Name: "Extras", NameTe: "అదనపు", Min: 0, Max: 3, Addons: []addon{
				{"Extra Raita", "అదనపు రైతా", 20, veg},
				{"Mirchi ka Salan", "మిర్చి కా సాలన్", 25, veg},
				{"Boiled Egg", "ఉడికించిన గుడ్డు", 15, egg},
				{"Chicken 65 (4 pc)", "చికెన్ 65 (4 ముక్కలు)", 90, nonVeg},
			}},
			{Key: "drink", Name: "Add a drink", NameTe: "పానీయం జోడించండి", Min: 0, Max: 1, Addons: []addon{
				{"Buttermilk", "మజ్జిగ", 25, veg},
				{"Sweet Lassi", "స్వీట్ లస్సీ", 45, veg},
			}},
		},
		Categories: []category{
			{"Biryani", "బిర్యానీ", []item{
				{Name: "Chicken Dum Biryani", NameTe: "చికెన్ దమ్ బిర్యానీ", Desc: "Slow-cooked basmati rice with marinated chicken.", Veg: nonVeg, Rupees: 240, PackagingRupees: 10, Variants: halfFull(150, 240), AddonGroups: []string{"extras", "drink"}, Recommended: true, Spice: 2},
				{Name: "Mutton Dum Biryani", NameTe: "మటన్ దమ్ బిర్యానీ", Desc: "Tender mutton layered with fragrant rice.", Veg: nonVeg, Rupees: 340, PackagingRupees: 10, Variants: halfFull(210, 340), AddonGroups: []string{"extras", "drink"}, Spice: 2},
				{Name: "Egg Biryani", NameTe: "ఎగ్ బిర్యానీ", Desc: "Biryani rice with two boiled eggs.", Veg: egg, Rupees: 170, PackagingRupees: 10, Variants: halfFull(110, 170), AddonGroups: []string{"extras", "drink"}, Spice: 2},
				{Name: "Veg Dum Biryani", NameTe: "వెజ్ దమ్ బిర్యానీ", Desc: "Seasonal vegetables and paneer in dum rice.", Veg: veg, Rupees: 180, PackagingRupees: 10, Variants: halfFull(120, 180), AddonGroups: []string{"extras", "drink"}, Spice: 1},
			}},
			{"Starters", "స్టార్టర్స్", []item{
				{Name: "Chicken 65", NameTe: "చికెన్ 65", Desc: "Crispy fried chicken tossed with curry leaves.", Veg: nonVeg, Rupees: 190, PackagingRupees: 5, Recommended: true, Spice: 3},
				{Name: "Paneer Tikka", NameTe: "పనీర్ టిక్కా", Desc: "Char-grilled paneer with mint chutney.", Veg: veg, Rupees: 210, PackagingRupees: 5, Spice: 1},
				{Name: "Chicken Seekh Kebab", NameTe: "చికెన్ సీఖ్ కబాబ్", Desc: "Minced chicken skewers, four pieces.", Veg: nonVeg, Rupees: 220, PackagingRupees: 5, Spice: 2},
			}},
			{"Breads & Curries", "రోటీలు & కూరలు", []item{
				{Name: "Butter Naan", NameTe: "బటర్ నాన్", Veg: veg, Rupees: 40},
				{Name: "Butter Chicken", NameTe: "బటర్ చికెన్", Desc: "Chicken in a creamy tomato gravy.", Veg: nonVeg, Rupees: 260, PackagingRupees: 10, Spice: 1},
				{Name: "Paneer Butter Masala", NameTe: "పనీర్ బటర్ మసాలా", Veg: veg, Rupees: 220, PackagingRupees: 10, Spice: 1},
			}},
			{"Desserts", "డెజర్ట్స్", []item{
				{Name: "Double ka Meetha", NameTe: "డబుల్ కా మీఠా", Desc: "Bread pudding with saffron and nuts.", Veg: veg, Rupees: 80, PackagingRupees: 5},
				{Name: "Qubani ka Meetha", NameTe: "ఖుబానీ కా మీఠా", Desc: "Stewed apricots with cream.", Veg: veg, Rupees: 90, PackagingRupees: 5},
			}},
		},
	},
	{
		Slug: "demo-tiffin-centre", Name: "Demo Tiffin Centre", NameTe: "డెమో టిఫిన్ సెంటర్",
		Desc: "Idli, dosa, vada and filter coffee, pure vegetarian. Fictional demo outlet.", Diet: "PURE_VEG",
		Lat: 16.7300, Lng: 77.9800, Locality: "boyapalle",
		Address: "Plot 12, Demo Road, Boyapalle", Landmark: "Opposite the demo bus stop",
		Phone: "+919999900002", OwnerPhone: "+919999900012", OwnerName: "Demo Owner Sunitha",
		Cuisines: []string{"TIFFINS", "SOUTH_INDIAN", "TEA_COFFEE"}, PrepMin: 15, CostForTwoRupees: 150, RadiusM: 7000,
		Opens: "06:30", Closes: "21:30",
		AddonGroups: []addonGroup{
			{Key: "chutney", Name: "Extra chutney & podi", NameTe: "అదనపు చట్నీ & పొడి", Min: 0, Max: 3, Addons: []addon{
				{"Extra Coconut Chutney", "అదనపు కొబ్బరి చట్నీ", 10, veg},
				{"Extra Sambar", "అదనపు సాంబార్", 15, veg},
				{"Karam Podi with Ghee", "కారం పొడి నెయ్యితో", 20, veg},
			}},
			{Key: "dosa-style", Name: "Choose your roast", NameTe: "రోస్ట్ ఎంచుకోండి", Min: 1, Max: 1, Addons: []addon{
				{"Regular", "రెగ్యులర్", 0, veg},
				{"Ghee Roast", "నెయ్యి రోస్ట్", 20, veg},
				{"Butter Roast", "బటర్ రోస్ట్", 20, veg},
			}},
		},
		Categories: []category{
			{"Idli & Vada", "ఇడ్లీ & వడ", []item{
				{Name: "Idli", NameTe: "ఇడ్లీ", Desc: "Steamed rice cakes with chutney and sambar.", Veg: veg, Rupees: 40, PackagingRupees: 5,
					Variants: []variant{{"2 pieces", "2 ముక్కలు", 30, false}, {"4 pieces", "4 ముక్కలు", 40, true}}, AddonGroups: []string{"chutney"}, Recommended: true},
				{Name: "Medu Vada", NameTe: "మెదు వడ", Desc: "Crisp lentil doughnuts, two pieces.", Veg: veg, Rupees: 45, PackagingRupees: 5, AddonGroups: []string{"chutney"}},
				{Name: "Ghee Karam Idli", NameTe: "నెయ్యి కారం ఇడ్లీ", Desc: "Idli tossed in ghee and spiced podi.", Veg: veg, Rupees: 60, PackagingRupees: 5, Spice: 2},
			}},
			{"Dosa", "దోశ", []item{
				{Name: "Plain Dosa", NameTe: "ప్లెయిన్ దోశ", Veg: veg, Rupees: 50, PackagingRupees: 5, AddonGroups: []string{"dosa-style", "chutney"}},
				{Name: "Masala Dosa", NameTe: "మసాలా దోశ", Desc: "Dosa filled with spiced potato.", Veg: veg, Rupees: 70, PackagingRupees: 5, AddonGroups: []string{"dosa-style", "chutney"}, Recommended: true, Spice: 1},
				{Name: "Onion Dosa", NameTe: "ఉల్లి దోశ", Veg: veg, Rupees: 65, PackagingRupees: 5, AddonGroups: []string{"dosa-style", "chutney"}, Spice: 1},
				{Name: "Pesarattu Upma", NameTe: "పెసరట్టు ఉప్మా", Desc: "Green-gram dosa stuffed with upma.", Veg: veg, Rupees: 80, PackagingRupees: 5, AddonGroups: []string{"chutney"}, Spice: 1},
			}},
			{"Rice & More", "అన్నం & ఇతరాలు", []item{
				{Name: "Pulihora", NameTe: "పులిహోర", Desc: "Tamarind rice with peanuts.", Veg: veg, Rupees: 60, PackagingRupees: 5, Spice: 1},
				{Name: "Curd Rice", NameTe: "పెరుగన్నం", Veg: veg, Rupees: 55, PackagingRupees: 5},
				{Name: "Poori Curry", NameTe: "పూరీ కూర", Desc: "Three pooris with potato curry.", Veg: veg, Rupees: 60, PackagingRupees: 5},
			}},
			{"Beverages", "పానీయాలు", []item{
				{Name: "Filter Coffee", NameTe: "ఫిల్టర్ కాఫీ", Veg: veg, Rupees: 25, PackagingRupees: 3},
				{Name: "Irani Chai", NameTe: "ఇరానీ చాయ్", Veg: veg, Rupees: 20, PackagingRupees: 3},
			}},
		},
	},
	{
		Slug: "demo-telangana-meals", Name: "Demo Telangana Meals", NameTe: "డెమో తెలంగాణ భోజనం",
		Desc: "Home-style Telangana meals, curries and jonna rotte. Fictional demo outlet.", Diet: "MIXED",
		Lat: 16.7600, Lng: 78.0300, Locality: "padmavathi-colony",
		Address: "Door 4-5, Demo Street, Padmavathi Colony", Landmark: "Beside the demo park",
		Phone: "+919999900003", OwnerPhone: "+919999900013", OwnerName: "Demo Owner Narsimha",
		Cuisines: []string{"ANDHRA_TELANGANA_MEALS", "SOUTH_INDIAN"}, PrepMin: 20, CostForTwoRupees: 300, RadiusM: 3000,
		Opens: "11:30", Closes: "22:00",
		AddonGroups: []addonGroup{
			{Key: "meal-extras", Name: "Add to your meal", NameTe: "మీ భోజనానికి జోడించండి", Min: 0, Max: 4, Addons: []addon{
				{"Extra Rice", "అదనపు అన్నం", 30, veg},
				{"Omelette", "ఆమ్లెట్", 30, egg},
				{"Chicken Fry (small)", "చికెన్ ఫ్రై (చిన్నది)", 90, nonVeg},
				{"Papad (2 pc)", "అప్పడం (2)", 10, veg},
			}},
		},
		Categories: []category{
			{"Meals", "భోజనం", []item{
				{Name: "Veg Meals", NameTe: "వెజ్ భోజనం", Desc: "Rice, pappu, two curries, sambar, rasam, curd and pickle.", Veg: veg, Rupees: 120, PackagingRupees: 10, AddonGroups: []string{"meal-extras"}, Recommended: true, Spice: 2},
				{Name: "Chicken Meals", NameTe: "చికెన్ భోజనం", Desc: "Veg meals with country-style chicken curry.", Veg: nonVeg, Rupees: 180, PackagingRupees: 10, AddonGroups: []string{"meal-extras"}, Spice: 3},
				{Name: "Egg Meals", NameTe: "ఎగ్ భోజనం", Desc: "Veg meals with egg curry.", Veg: egg, Rupees: 140, PackagingRupees: 10, AddonGroups: []string{"meal-extras"}, Spice: 2},
			}},
			{"Curries", "కూరలు", []item{
				{Name: "Natu Kodi Pulusu", NameTe: "నాటు కోడి పులుసు", Desc: "Country chicken in a spicy tamarind gravy.", Veg: nonVeg, Rupees: 260, PackagingRupees: 10, Variants: halfFull(160, 260), Recommended: true, Spice: 3},
				{Name: "Mutton Curry", NameTe: "మటన్ కూర", Veg: nonVeg, Rupees: 320, PackagingRupees: 10, Variants: halfFull(190, 320), Spice: 3},
				{Name: "Gutti Vankaya Kura", NameTe: "గుత్తి వంకాయ కూర", Desc: "Stuffed brinjal curry.", Veg: veg, Rupees: 130, PackagingRupees: 10, Spice: 2},
				{Name: "Pappu Tomato", NameTe: "టమాటా పప్పు", Veg: veg, Rupees: 90, PackagingRupees: 10, Spice: 1},
				{Name: "Egg Pulusu", NameTe: "గుడ్డు పులుసు", Veg: egg, Rupees: 110, PackagingRupees: 10, Spice: 2},
			}},
			{"Rotte", "రొట్టె", []item{
				{Name: "Jonna Rotte", NameTe: "జొన్న రొట్టె", Desc: "Hand-patted sorghum flatbread.", Veg: veg, Rupees: 25},
				{Name: "Sarva Pindi", NameTe: "సర్వ పిండి", Desc: "Crisp rice-flour pancake with peanuts and chana dal.", Veg: veg, Rupees: 60, PackagingRupees: 5, Spice: 1},
			}},
		},
	},
	{
		Slug: "demo-chinese-corner", Name: "Demo Chinese Corner", NameTe: "డెమో చైనీస్ కార్నర్",
		Desc: "Indo-Chinese noodles, fried rice and manchurian. Fictional demo outlet.", Diet: "MIXED",
		Lat: 16.7400, Lng: 78.0150, Locality: "rajendra-nagar",
		Address: "Stall 7, Demo Market, Rajendra Nagar", Landmark: "Near the demo petrol pump",
		Phone: "+919999900004", OwnerPhone: "+919999900014", OwnerName: "Demo Owner Kavitha",
		Cuisines: []string{"INDO_CHINESE", "FAST_FOOD"}, PrepMin: 20, CostForTwoRupees: 250, RadiusM: 7000,
		Opens: "12:00", Closes: "23:00",
		AddonGroups: []addonGroup{
			{Key: "spice", Name: "Spice level", NameTe: "కారం స్థాయి", Min: 1, Max: 1, Addons: []addon{
				{"Mild", "తక్కువ కారం", 0, veg},
				{"Medium", "మధ్యస్థ కారం", 0, veg},
				{"Spicy", "ఎక్కువ కారం", 0, veg},
			}},
			{Key: "toppings", Name: "Toppings", NameTe: "టాపింగ్స్", Min: 0, Max: 2, Addons: []addon{
				{"Fried Egg", "ఫ్రైడ్ ఎగ్", 20, egg},
				{"Extra Chicken", "అదనపు చికెన్", 50, nonVeg},
				{"Extra Paneer", "అదనపు పనీర్", 40, veg},
			}},
		},
		Categories: []category{
			{"Noodles", "నూడుల్స్", []item{
				{Name: "Veg Hakka Noodles", NameTe: "వెజ్ హక్కా నూడుల్స్", Veg: veg, Rupees: 110, PackagingRupees: 8, Variants: halfFull(70, 110), AddonGroups: []string{"spice", "toppings"}},
				{Name: "Egg Noodles", NameTe: "ఎగ్ నూడుల్స్", Veg: egg, Rupees: 130, PackagingRupees: 8, Variants: halfFull(80, 130), AddonGroups: []string{"spice", "toppings"}},
				{Name: "Chicken Schezwan Noodles", NameTe: "చికెన్ షెజ్వాన్ నూడుల్స్", Veg: nonVeg, Rupees: 160, PackagingRupees: 8, Variants: halfFull(100, 160), AddonGroups: []string{"spice", "toppings"}, Recommended: true, Spice: 3},
			}},
			{"Fried Rice", "ఫ్రైడ్ రైస్", []item{
				{Name: "Veg Fried Rice", NameTe: "వెజ్ ఫ్రైడ్ రైస్", Veg: veg, Rupees: 110, PackagingRupees: 8, Variants: halfFull(70, 110), AddonGroups: []string{"spice", "toppings"}},
				{Name: "Egg Fried Rice", NameTe: "ఎగ్ ఫ్రైడ్ రైస్", Veg: egg, Rupees: 130, PackagingRupees: 8, Variants: halfFull(80, 130), AddonGroups: []string{"spice", "toppings"}, Recommended: true},
				{Name: "Chicken Fried Rice", NameTe: "చికెన్ ఫ్రైడ్ రైస్", Veg: nonVeg, Rupees: 150, PackagingRupees: 8, Variants: halfFull(95, 150), AddonGroups: []string{"spice", "toppings"}},
			}},
			{"Starters", "స్టార్టర్స్", []item{
				{Name: "Gobi Manchurian", NameTe: "గోబీ మంచూరియా", Desc: "Crispy cauliflower in a tangy sauce.", Veg: veg, Rupees: 120, PackagingRupees: 8, AddonGroups: []string{"spice"}, Spice: 2},
				{Name: "Chilli Chicken", NameTe: "చిల్లీ చికెన్", Veg: nonVeg, Rupees: 180, PackagingRupees: 8, AddonGroups: []string{"spice"}, Spice: 3},
				{Name: "Veg Spring Rolls", NameTe: "వెజ్ స్ప్రింగ్ రోల్స్", Desc: "Four pieces with sweet chilli dip.", Veg: veg, Rupees: 100, PackagingRupees: 8},
			}},
			{"Soups", "సూప్స్", []item{
				{Name: "Sweet Corn Soup", NameTe: "స్వీట్ కార్న్ సూప్", Veg: veg, Rupees: 70, PackagingRupees: 8},
				{Name: "Chicken Hot & Sour Soup", NameTe: "చికెన్ హాట్ & సోర్ సూప్", Veg: nonVeg, Rupees: 90, PackagingRupees: 8, Spice: 2},
			}},
		},
	},
}
