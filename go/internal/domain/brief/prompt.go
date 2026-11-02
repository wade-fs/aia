package brief

import (
	"fmt"
	"strings"
)

// BriefAnalysisJSONSchema defines the formal JSON Schema specification for agy output (AIA-21 §6.2).
const BriefAnalysisJSONSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "AgentBriefAnalysis",
  "type": "object",
  "required": ["summary", "understanding", "missing_fields", "clarification_questions", "proposed_defaults", "research_candidates", "readiness"],
  "properties": {
    "summary": { "type": "string", "description": "High-level summary of the project goal and current understanding" },
    "understanding": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Key takeaways of the user's intent and expectations"
    },
    "missing_fields": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["field", "importance", "blocking", "reason"],
        "properties": {
          "field": { "type": "string" },
          "importance": { "type": "string", "enum": ["high", "medium", "low"] },
          "blocking": { "type": "boolean" },
          "reason": { "type": "string" }
        }
      }
    },
    "clarification_questions": {
      "type": "array",
      "maxItems": 5,
      "items": {
        "type": "object",
        "required": ["field_key", "question", "reason", "impact", "priority", "blocking"],
        "properties": {
          "field_key": { "type": "string" },
          "question": { "type": "string" },
          "reason": { "type": "string" },
          "impact": { "type": "string" },
          "options": { "type": "array", "items": { "type": "string" } },
          "priority": { "type": "integer", "enum": [0, 1, 2, 3] },
          "blocking": { "type": "boolean" }
        }
      }
    },
    "proposed_defaults": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["field", "value", "rationale", "confidence"],
        "properties": {
          "field": { "type": "string" },
          "value": {},
          "rationale": { "type": "string" },
          "confidence": { "type": "number", "minimum": 0.0, "maximum": 0.99 }
        }
      }
    },
    "research_candidates": {
      "type": "array",
      "items": { "type": "string" }
    },
    "recommendations": {
      "type": "array",
      "items": { "type": "string" }
    },
    "readiness": {
      "type": "object",
      "required": ["status", "reason"],
      "properties": {
        "status": { "type": "string", "enum": ["NOT_READY", "READY_WITH_ASSUMPTIONS", "READY_FOR_RESEARCH", "READY_FOR_PLANNING", "READY_FOR_EXECUTION", "NEEDS_REASSESSMENT"] },
        "reason": { "type": "string" }
      }
    }
  }
}`

// BuildBriefAnalysisPrompt generates the instruction prompt for agy to analyze a Project Brief (AIA-21 §6, §9).
func BuildBriefAnalysisPrompt(
	brief *ProjectBrief,
	catalog *DomainCatalog,
	agentInstructions string,
	memorySnippets []string,
) string {
	var sb strings.Builder

	sb.WriteString("=== AGENT BRIEF ANALYSIS MISSION (AIA-21) ===\n")
	sb.WriteString("You are the Domain Specialist Agent responsible for evaluating requirements, detecting specification gaps, and formulating high-value clarification questions.\n\n")

	if agentInstructions != "" {
		sb.WriteString("## DOMAIN AGENT PROFILE\n")
		sb.WriteString(agentInstructions)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## PROJECT SPECIFICATION\n")
	sb.WriteString(fmt.Sprintf("Project ID:   %s\n", brief.ProjectID))
	sb.WriteString(fmt.Sprintf("Domain:       %s\n", brief.Domain))
	sb.WriteString(fmt.Sprintf("Raw Intent:   %s\n", brief.RawIntent))
	sb.WriteString(fmt.Sprintf("Brief Version: v%d\n", brief.BriefVersion))
	sb.WriteString(fmt.Sprintf("Current Readiness: %s (%s)\n\n", brief.Readiness, brief.ReadinessRationale))

	// Serialize current brief YAML
	if briefYAML, err := brief.ToYAML(); err == nil {
		sb.WriteString("### CURRENT BRIEF (YAML)\n```yaml\n")
		sb.WriteString(briefYAML)
		sb.WriteString("\n```\n\n")
	}

	// Catalog fields
	if catalog != nil && len(catalog.Fields) > 0 {
		sb.WriteString("## AUTHORIZED DOMAIN CATALOG FIELDS\n")
		sb.WriteString("Notice: You MUST ONLY refer to the following authorized field keys. Do NOT invent new field keys.\n\n")
		for key, def := range catalog.Fields {
			reqStr := "Supplementary"
			if def.RequiredForPlanning {
				reqStr = "REQUIRED_FOR_PLANNING"
			}
			defValStr := "none"
			if def.DefaultValue != nil {
				defValStr = fmt.Sprintf("%v", def.DefaultValue)
			}
			sb.WriteString(fmt.Sprintf("- **%s** (%s, %s): %s (Default: %s)\n",
				key, def.Importance, reqStr, def.Description, defValStr))
		}
		sb.WriteString("\n")
	}

	if len(memorySnippets) > 0 {
		sb.WriteString("## PROJECT MEMORY & DECISIONS\n")
		for _, m := range memorySnippets {
			sb.WriteString(fmt.Sprintf("- %s\n", m))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## ANALYSIS INSTRUCTIONS & DISCIPLINE RULES\n")
	sb.WriteString("1. **Understand Intent**: Analyze both explicit user statements and downstream structural needs.\n")
	sb.WriteString("2. **Identify Unknowns**: Check which authorized catalog fields are missing or UNKNOWN.\n")
	sb.WriteString("3. **Question Discipline**: Do NOT overwhelm the user. Formulate at most 5 high-impact questions (P0/P1 only). If only 1 or 2 are truly needed, ask only those.\n")
	sb.WriteString("4. **Mandatory Reasons & Impact**: Every question MUST articulate *why* it is asked (decision reason) and *what* downstream work it affects (architecture impact).\n")
	sb.WriteString("5. **Propose Defaults**: For operational details that do not block overall planning, propose reasonable defaults with clear rationale. Never assume defaults are confirmed facts (confidence < 1.0).\n")
	sb.WriteString("6. **Identify Research**: Note items that should be investigated by the agent rather than burdening the user.\n")
	sb.WriteString("7. **Output Format**: Respond ONLY with a valid JSON object conforming to the schema below. No markdown fences around the raw JSON output if possible, or a single json block.\n\n")

	sb.WriteString("## REQUIRED JSON SCHEMA\n")
	sb.WriteString(BriefAnalysisJSONSchema)
	sb.WriteString("\n")

	return sb.String()
}
