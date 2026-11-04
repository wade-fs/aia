package knowledge

import "time"

type SourceType string

const (
	SourcePrimary   SourceType = "PRIMARY"
	SourceOfficial  SourceType = "OFFICIAL"
	SourceSecondary SourceType = "SECONDARY"
	SourceNews      SourceType = "NEWS"
	SourceAcademic  SourceType = "ACADEMIC"
	SourceCommunity SourceType = "COMMUNITY"
	SourceModel     SourceType = "MODEL"
	SourceTool      SourceType = "TOOL"
)

// Source represents an external or internal origin of information with 5-dimensional reliability.
type Source struct {
	ID                   string     `json:"id"`
	SourceType           SourceType `json:"source_type"`
	URI                  string     `json:"uri"`
	Publisher            string     `json:"publisher,omitempty"`
	AuthorityScore       float64    `json:"authority_score"`       // 0.0 ~ 1.0 (official/academic vs anonymous)
	DirectnessScore      float64    `json:"directness_score"`      // 0.0 ~ 1.0 (first-hand vs repost)
	TimelinessScore      float64    `json:"timeliness_score"`      // 0.0 ~ 1.0 (freshness)
	IndependenceScore    float64    `json:"independence_score"`    // 0.0 ~ 1.0 (independent vs sponsored/PR)
	ReproducibilityScore float64    `json:"reproducibility_score"` // 0.0 ~ 1.0 (repeatable measurement)
	MetadataJSON         string     `json:"metadata_json"`
	CreatedAt            time.Time  `json:"created_at"`
}

type CaptureMethod string

const (
	CaptureHTTPGet    CaptureMethod = "HTTP_GET"
	CaptureHTMLParsed CaptureMethod = "HTML_PARSED"
	CaptureAPIJSON    CaptureMethod = "API_JSON"
	CapturePDFOCR     CaptureMethod = "PDF_OCR"
	CaptureToolStdout CaptureMethod = "TOOL_STDOUT"
)

// EvidenceStatus defines the lifecycle status of evidence (AIA-16 §4.2).
type EvidenceStatus string

const (
	EvidenceCollected  EvidenceStatus = "COLLECTED"
	EvidenceParsed     EvidenceStatus = "PARSED"
	EvidenceNormalized EvidenceStatus = "NORMALIZED"
	EvidenceValidated  EvidenceStatus = "VALIDATED"
	EvidenceReviewed   EvidenceStatus = "REVIEWED"
	EvidenceVerified   EvidenceStatus = "VERIFIED"
	EvidenceRejected   EvidenceStatus = "REJECTED"
	EvidenceUncertain  EvidenceStatus = "UNCERTAIN"
	EvidenceConflicted EvidenceStatus = "CONFLICTED"
	EvidenceDeprecated EvidenceStatus = "DEPRECATED"
)

// Evidence represents an excerpt or observation grounded in a specific Source.
type Evidence struct {
	ID             string         `json:"id"`
	SourceID       string         `json:"source_id"`
	ContentSnippet string         `json:"content_snippet"`
	ContentHash    string         `json:"content_hash"`
	CaptureMethod  CaptureMethod  `json:"capture_method"`
	DataVersion    string         `json:"data_version"`
	Status         EvidenceStatus `json:"status,omitempty"`
	PublishedAt    *time.Time     `json:"published_at,omitempty"`
	CapturedAt     time.Time      `json:"captured_at"`
	ValidUntil     *time.Time     `json:"valid_until,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// ClaimType defines the epistemic categorization of a proposition (AIA-16 §7.1 & §12.3.2).
type ClaimType string

const (
	ClaimFact                ClaimType = "FACT"
	ClaimInference           ClaimType = "INFERENCE"
	ClaimHypothesis          ClaimType = "HYPOTHESIS"
	ClaimOpinion             ClaimType = "OPINION"
	ClaimModelOutput         ClaimType = "MODEL_OUTPUT"
	ClaimUserAssertion       ClaimType = "USER_ASSERTION"
	ClaimHistoricalSkeleton  ClaimType = "HISTORICAL_SKELETON"
	ClaimReasonableInference ClaimType = "REASONABLE_INFERENCE"
	ClaimFictionalConstruct  ClaimType = "FICTIONAL_CONSTRUCT"
	ClaimUnknown             ClaimType = "UNKNOWN"
)

type AssertionStatus string

const (
	AssertionAccepted   AssertionStatus = "ACCEPTED"
	AssertionRejected   AssertionStatus = "REJECTED"
	AssertionUncertain  AssertionStatus = "UNCERTAIN"
	AssertionConflicted AssertionStatus = "CONFLICTED"
	AssertionSuperseded AssertionStatus = "SUPERSEDED"
)

// Claim represents a testable proposition or assertion.
type Claim struct {
	ID              string          `json:"id"`
	Statement       string          `json:"statement"`
	ClaimType       ClaimType       `json:"claim_type,omitempty"`
	AssertionStatus AssertionStatus `json:"assertion_status"`
	IsKnowledge     bool            `json:"is_knowledge"`
	CurrentVersion  int             `json:"current_version"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ClaimRevision preserves historical statements and reasons for revision.
type ClaimRevision struct {
	ID                string    `json:"id"`
	ClaimID           string    `json:"claim_id"`
	RevisionNumber    int       `json:"revision_number"`
	PreviousStatement string    `json:"previous_statement"`
	NewStatement      string    `json:"new_statement"`
	Reason            string    `json:"reason"`
	RevisedDecisionID string    `json:"revised_decision_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type SupportType string

const (
	SupportTypeSupports       SupportType = "SUPPORTS"
	SupportTypeRefutes        SupportType = "REFUTES"
	SupportTypeContradicts    SupportType = "CONTRADICTS"
	SupportTypeQualifies      SupportType = "QUALIFIES"
	SupportTypeContextualizes SupportType = "CONTEXTUALIZES"
	SupportTypeDerivedFrom    SupportType = "DERIVED_FROM"
	SupportTypeDoesNotAddress SupportType = "DOES_NOT_ADDRESS"
)

// ClaimEvidenceLink associates a Claim with supporting or refuting Evidence.
type ClaimEvidenceLink struct {
	ClaimID          string      `json:"claim_id"`
	EvidenceID       string      `json:"evidence_id"`
	SupportType      SupportType `json:"support_type"`
	ConfidenceWeight float64     `json:"confidence_weight"`
}

type ProvenanceType string

const (
	ProvenanceFactBased      ProvenanceType = "FACT_BASED"
	ProvenanceInference      ProvenanceType = "INFERENCE"
	ProvenanceHypothesis     ProvenanceType = "HYPOTHESIS"
	ProvenanceUserPreference ProvenanceType = "USER_PREFERENCE"
	ProvenanceAgentDecision  ProvenanceType = "AGENT_DECISION"
)

type ProvenanceRefType string

const (
	ProvenanceRefEvidence        ProvenanceRefType = "EVIDENCE"
	ProvenanceRefSource          ProvenanceRefType = "SOURCE"
	ProvenanceRefMemory          ProvenanceRefType = "MEMORY"
	ProvenanceRefContextSnapshot ProvenanceRefType = "CONTEXT_SNAPSHOT"
	ProvenanceRefDecision        ProvenanceRefType = "DECISION"
	ProvenanceRefNone            ProvenanceRefType = "NONE"
)

type ProvenanceStatus string

const (
	ProvenanceStatusVerified   ProvenanceStatus = "VERIFIED"
	ProvenanceStatusPartial    ProvenanceStatus = "PARTIAL"
	ProvenanceStatusUnverified ProvenanceStatus = "UNVERIFIED"
)
