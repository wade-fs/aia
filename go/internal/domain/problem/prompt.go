package problem

import (
	"encoding/json"
	"fmt"
	"strings"

	domainProject "github.com/wade/aia/internal/domain/project"
)

// BuildProblemAnalysisPrompt constructs an epistemic problem analysis prompt for an Agent (AIA-22 §4).
func BuildProblemAnalysisPrompt(
	p *Problem,
	proj *domainProject.Project,
	userPrompt string,
	contextFindings []string,
	manifestRules []string,
) string {
	var b strings.Builder

	b.WriteString("# AIA-22 Epistemic Problem Analysis & Modeling\n\n")
	b.WriteString("你正在執行 AIA 複雜未知問題分析。面對開放式或尚未充分定義之目標，你必須自主探索問題背景、建立問題模型、發現潛在缺口與規範衝突，而非直接盲目拆解 Task。\n\n")

	b.WriteString("## 1. 專案與問題上下文 (Target Context)\n")
	if p != nil {
		b.WriteString(fmt.Sprintf("- **Problem ID**: %s\n", p.ID))
		b.WriteString(fmt.Sprintf("- **Problem Title**: %s\n", p.Title))
		b.WriteString(fmt.Sprintf("- **Objective**: %s\n", p.Objective))
		if p.Scope != "" {
			b.WriteString(fmt.Sprintf("- **Scope**: %s\n", p.Scope))
		}
		if p.ParentID != "" {
			b.WriteString(fmt.Sprintf("- **Parent Problem**: %s\n", p.ParentID))
		}
	}
	if proj != nil {
		b.WriteString(fmt.Sprintf("- **Project ID**: %s\n", proj.ID))
		b.WriteString(fmt.Sprintf("- **Project Name**: %s\n", proj.Name))
		b.WriteString(fmt.Sprintf("- **Domain**: %s\n", proj.DomainType))
		b.WriteString(fmt.Sprintf("- **Workspace**: %s\n", proj.Workspace))
	}
	b.WriteString(fmt.Sprintf("- **User Raw Input**: %s\n\n", userPrompt))

	if len(contextFindings) > 0 {
		b.WriteString("## 2. 現況調查線索 (Context Findings)\n")
		for _, f := range contextFindings {
			b.WriteString(fmt.Sprintf("- %s\n", f))
		}
		b.WriteString("\n")
	}

	if len(manifestRules) > 0 {
		b.WriteString("## 3. 現有規範與約束文件 (Manifest Rules)\n")
		for _, r := range manifestRules {
			b.WriteString(fmt.Sprintf("- %s\n", r))
		}
		b.WriteString("\n")
	}

	b.WriteString("## 4. 分析核心原則 (Epistemic Principles, AIA-22 §1.2, §7)\n")
	b.WriteString("1. **事實與推論分離 (Fact vs Inference)**：\n")
	b.WriteString("   - `KNOWN_FACT`：必須有明確可查證之來源（如代碼行數、文件內容），Confidence 必須為 1.0。\n")
	b.WriteString("   - `ASSUMPTION`：為了推進分析暫時採用的假設，Confidence 必須 < 1.0，嚴禁假裝已確認。\n")
	b.WriteString("   - `UNKNOWN`：尚未掌握的事項，Confidence 必須為 0.0。\n")
	b.WriteString("2. **意圖邊界控制 (Intent Boundary)**：\n")
	b.WriteString("   - 若使用者僅要求「分析」、「檢查」或「評估」，不可擅自轉為修改或執行 (`analysis_only: true`)。\n")
	b.WriteString("3. **主動發掘缺口與衝突 (Gap & Conflict Discovery)**：\n")
	b.WriteString("   - 尋找隱藏依賴、資料缺口、規範衝突與潛在風險。\n")
	b.WriteString("4. **人機詢問紀律 (Clarification Discipline, AIA-21 §5.4)**：\n")
	b.WriteString("   - 能自行藉由閱讀文件、代碼取得答案者，不可詢問使用者。\n")
	b.WriteString("   - 若必須提問（涉及人類意圖、重大取捨），單批提問不得超過 5 題，且每題必須具備決策價值 (Reason) 與影響 (Impact)。\n\n")

	b.WriteString("## 5. 輸出格式 (JSON Output Schema)\n")
	b.WriteString("請嚴格輸出符合 Draft-07 JSON Schema 的 JSON 物件，包含 intent、context、gaps_and_conflicts、model_items 與 clarification_questions。\n")

	return b.String()
}

// ProblemAnalysisDraft07Schema returns the JSON Schema for AgentProblemAnalysis.
func ProblemAnalysisDraft07Schema() map[string]interface{} {
	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "AgentProblemAnalysis",
		"type": "object",
		"required": ["intent", "context", "model_items"],
		"properties": {
			"intent": {
				"type": "object",
				"required": ["user_goal"],
				"properties": {
					"user_goal": { "type": "string" },
					"requested_method": { "type": "string" },
					"constraints": { "type": "array", "items": { "type": "string" } },
					"assumptions": { "type": "array", "items": { "type": "string" } },
					"success_criteria": { "type": "array", "items": { "type": "string" } },
					"analysis_only": { "type": "boolean" }
				}
			},
			"context": {
				"type": "object",
				"properties": {
					"found_resources": { "type": "array", "items": { "type": "string" } },
					"missing_resources": { "type": "array", "items": { "type": "string" } },
					"trusted_data": { "type": "array", "items": { "type": "string" } },
					"needs_verification": { "type": "array", "items": { "type": "string" } },
					"conflicting_data": { "type": "array", "items": { "type": "string" } }
				}
			},
			"gaps_and_conflicts": {
				"type": "array",
				"items": {
					"type": "object",
					"required": ["id", "category", "description", "suggested_handling"],
					"properties": {
						"id": { "type": "string" },
						"category": { "type": "string" },
						"description": { "type": "string" },
						"source": { "type": "string" },
						"impact": { "type": "string" },
						"severity": { "type": "string" },
						"confidence": { "type": "number", "minimum": 0, "maximum": 1 },
						"suggested_handling": { "type": "string" },
						"requires_clarification": { "type": "boolean" }
					}
				}
			},
			"model_items": {
				"type": "array",
				"items": {
					"type": "object",
					"required": ["id", "category", "statement", "confidence"],
					"properties": {
						"id": { "type": "string" },
						"category": { "type": "string" },
						"statement": { "type": "string" },
						"source": { "type": "string" },
						"evidence_ref": { "type": "string" },
						"confidence": { "type": "number", "minimum": 0, "maximum": 1 },
						"status": { "type": "string" },
						"created_by": { "type": "string" }
					}
				}
			},
			"clarification_questions": {
				"type": "array",
				"maxItems": 5,
				"items": {
					"type": "object",
					"required": ["field_key", "question", "reason", "impact"],
					"properties": {
						"field_key": { "type": "string" },
						"question": { "type": "string" },
						"options": { "type": "array", "items": { "type": "string" } },
						"reason": { "type": "string" },
						"impact": { "type": "string" },
						"priority": { "type": "integer" },
						"blocking": { "type": "boolean" }
					}
				}
			}
		}
	}`

	var schema map[string]interface{}
	_ = json.Unmarshal([]byte(schemaJSON), &schema)
	return schema
}
