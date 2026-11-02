package brief

import "fmt"

// FieldImportance defines the prioritization of fields (AIA-21 §5.3).
type FieldImportance string

const (
	ImportanceP0 FieldImportance = "P0" // Safety, legal, critical direction, irreversible
	ImportanceP1 FieldImportance = "P1" // Core architecture, major planning parameter
	ImportanceP2 FieldImportance = "P2" // Local layout, secondary arrangement
	ImportanceP3 FieldImportance = "P3" // Non-blocking / aesthetic
)

// FieldDefinition specifies the schema, constraints, and requirements of a single domain field.
type FieldDefinition struct {
	Key                  string          `json:"key"`
	Description          string          `json:"description"`
	Importance           FieldImportance `json:"importance"`
	RequiredForPlanning  bool            `json:"required_for_planning"`
	RequiredForExecution bool            `json:"required_for_execution"`
	DefaultValue         any             `json:"default_value,omitempty"`
	AllowedValues        []string        `json:"allowed_values,omitempty"`
	CanResearch          bool            `json:"can_research"` // Can Agent acquire info via research instead of asking?
}

// DomainCatalog contains the formal field definitions for a specific domain.
type DomainCatalog struct {
	Domain      string                     `json:"domain"`
	Description string                     `json:"description"`
	Fields      map[string]FieldDefinition `json:"fields"`
}

// Global registry of domain catalogs
var domainCatalogs = make(map[string]*DomainCatalog)

func init() {
	RegisterCatalog(NewNonfictionPlanCatalog())
	RegisterCatalog(NewHistoricalFictionCatalog())
	RegisterCatalog(NewCodingCatalog())
	RegisterCatalog(NewInvestmentCatalog())
}

// RegisterCatalog adds or overrides a domain catalog.
func RegisterCatalog(c *DomainCatalog) {
	domainCatalogs[c.Domain] = c
}

// GetCatalog retrieves a domain catalog by domain name.
func GetCatalog(domain string) (*DomainCatalog, error) {
	c, ok := domainCatalogs[domain]
	if !ok {
		return nil, fmt.Errorf("no domain catalog registered for domain '%s'", domain)
	}
	return c, nil
}

// NewNonfictionPlanCatalog defines specifications for real-world activity & planning books (AIA-21 §4.3 & §17).
func NewNonfictionPlanCatalog() *DomainCatalog {
	fields := []FieldDefinition{
		{Key: "activity", Description: "計畫的實際活動項目 (如: 徒步環台灣本島一圈)", Importance: ImportanceP0, RequiredForPlanning: true},
		{Key: "route_scope", Description: "路線範圍與方向 (順時針/逆時針)", Importance: ImportanceP0, RequiredForPlanning: true, AllowedValues: []string{"順時針", "逆時針", "雙向/分段"}},
		{Key: "start_location", Description: "徒步起點", Importance: ImportanceP1, RequiredForPlanning: true},
		{Key: "end_location", Description: "徒步終點", Importance: ImportanceP1, RequiredForPlanning: true},
		{Key: "duration_days", Description: "預計總天數", Importance: ImportanceP1, RequiredForPlanning: true},
		{Key: "daily_mileage_km", Description: "每日平均步行距離 (公里)", Importance: ImportanceP0, RequiredForPlanning: true, DefaultValue: 20},
		{Key: "transport_policy", Description: "是否允許部分路段搭乘大眾運輸 (如蘇花段替代)", Importance: ImportanceP0, RequiredForPlanning: true, AllowedValues: []string{"完全純徒步", "允許危險路段替代大眾運輸", "混合彈性交通"}},
		{Key: "safety_guidelines", Description: "安全守則與緊急醫療應變", Importance: ImportanceP0, RequiredForPlanning: true, CanResearch: true},
		{Key: "accommodation", Description: "住宿偏好 (露營/民宿/青年旅館/隨機)", Importance: ImportanceP1, RequiredForPlanning: false},
		{Key: "supplies_policy", Description: "裝備補給策略與重量上限", Importance: ImportanceP2, RequiredForPlanning: false, CanResearch: true},
		{Key: "budget_tier", Description: "預算級距", Importance: ImportanceP1, RequiredForPlanning: false},
		{Key: "target_audience", Description: "書籍目標讀者 (大眾/徒步愛好者/深度行者)", Importance: ImportanceP1, RequiredForPlanning: true},
		{Key: "book_type", Description: "書籍類型 (實用指南/旅行文學/個人計畫紀實/綜合作品)", Importance: ImportanceP1, RequiredForPlanning: true, AllowedValues: []string{"實用指南", "旅行文學", "計畫紀實", "綜合作品"}},
		{Key: "target_word_count", Description: "全書預計字數", Importance: ImportanceP1, RequiredForPlanning: true, DefaultValue: 120000},
	}

	fieldMap := make(map[string]FieldDefinition)
	for _, f := range fields {
		fieldMap[f.Key] = f
	}

	return &DomainCatalog{
		Domain:      "nonfiction_plan",
		Description: "非虛構專案規劃與實用紀實圖書 (如徒步環島指南)",
		Fields:      fieldMap,
	}
}

// NewHistoricalFictionCatalog defines specifications for historical fiction novels (AIA-20/21).
func NewHistoricalFictionCatalog() *DomainCatalog {
	fields := []FieldDefinition{
		{Key: "era", Description: "歷史年代與起訖年 (如: 元末 1328-1368)", Importance: ImportanceP0, RequiredForPlanning: true},
		{Key: "total_volumes", Description: "全書總卷數", Importance: ImportanceP1, RequiredForPlanning: true},
		{Key: "total_chapters", Description: "全書總章數", Importance: ImportanceP1, RequiredForPlanning: true},
		{Key: "chapters_per_volume", Description: "每卷章數", Importance: ImportanceP1, RequiredForPlanning: true},
		{Key: "core_protagonists", Description: "核心人物群像清單", Importance: ImportanceP0, RequiredForPlanning: true},
		{Key: "theme_philosophy", Description: "核心哲學原則與創作主旨", Importance: ImportanceP1, RequiredForPlanning: true},
		{Key: "worldview_rules", Description: "世界觀與知識分層規則 (史實/推演/虛構)", Importance: ImportanceP1, RequiredForPlanning: true},
	}

	fieldMap := make(map[string]FieldDefinition)
	for _, f := range fields {
		fieldMap[f.Key] = f
	}

	return &DomainCatalog{
		Domain:      "historical_fiction",
		Description: "長篇歷史群像小說創作",
		Fields:      fieldMap,
	}
}

// NewCodingCatalog defines specifications for coding tasks.
func NewCodingCatalog() *DomainCatalog {
	fields := []FieldDefinition{
		{Key: "target_repository", Description: "目標專案目錄與代碼庫位置", Importance: ImportanceP0, RequiredForPlanning: true},
		{Key: "language_runtime", Description: "程式語言與運行環境", Importance: ImportanceP0, RequiredForPlanning: true},
		{Key: "acceptance_criteria", Description: "驗收標準與測試用例", Importance: ImportanceP0, RequiredForPlanning: true, RequiredForExecution: true},
		{Key: "compatibility_targets", Description: "相容性目標與依賴約束", Importance: ImportanceP1, RequiredForPlanning: true},
	}

	fieldMap := make(map[string]FieldDefinition)
	for _, f := range fields {
		fieldMap[f.Key] = f
	}

	return &DomainCatalog{
		Domain:      "coding",
		Description: "程式碼開發與自主測試任務",
		Fields:      fieldMap,
	}
}

// NewInvestmentCatalog defines specifications for investment research and portfolios.
func NewInvestmentCatalog() *DomainCatalog {
	fields := []FieldDefinition{
		{Key: "investment_objective", Description: "投資目標與策略方向", Importance: ImportanceP0, RequiredForPlanning: true},
		{Key: "capital_budget", Description: "資金預算上限", Importance: ImportanceP0, RequiredForPlanning: true},
		{Key: "risk_tolerance", Description: "風險政策與停損門檻", Importance: ImportanceP0, RequiredForPlanning: true, RequiredForExecution: true},
		{Key: "benchmark_index", Description: "績效衡量基準指數", Importance: ImportanceP1, RequiredForPlanning: true},
	}

	fieldMap := make(map[string]FieldDefinition)
	for _, f := range fields {
		fieldMap[f.Key] = f
	}

	return &DomainCatalog{
		Domain:      "investment",
		Description: "量化研究與投資組合分析",
		Fields:      fieldMap,
	}
}
