package domain

type FeatureFlags struct {
	Observations      bool `json:"observations"`
	MCP               bool `json:"mcp"`
	Worker            bool `json:"worker"`
	BankConfigAPI     bool `json:"bank_config_api"`
	FileUploadAPI     bool `json:"file_upload_api"`
	DocumentExportAPI bool `json:"document_export_api"`
	DocumentImportAPI bool `json:"document_import_api"`
	AuditLog          bool `json:"audit_log"`
	LLMTrace          bool `json:"llm_trace"`
	StoreDocumentText bool `json:"store_document_text"`
}

type VersionInfo struct {
	APIVersion string       `json:"api_version"`
	Features   FeatureFlags `json:"features"`
}

type HealthStatus struct {
	OK      bool
	Version *VersionInfo
	Detail  string
}

type BankSummary struct {
	BankID         string      `json:"bank_id"`
	Name           *string     `json:"name"`
	Mission        *string     `json:"mission"`
	Disposition    Disposition `json:"disposition"`
	CreatedAt      *string     `json:"created_at"`
	UpdatedAt      *string     `json:"updated_at"`
	FactCount      int         `json:"fact_count"`
	LastDocumentAt *string     `json:"last_document_at"`
}

type Disposition struct {
	Skepticism int `json:"skepticism"`
	Literalism int `json:"literalism"`
	Empathy    int `json:"empathy"`
}

type BankProfile struct {
	BankID      string      `json:"bank_id"`
	Name        string      `json:"name"`
	Mission     string      `json:"mission"`
	Background  *string     `json:"background"`
	Disposition Disposition `json:"disposition"`
}

type BankConfig struct {
	BankID    string         `json:"bank_id"`
	Config    map[string]any `json:"config"`
	Overrides map[string]any `json:"overrides"`
}

type CreateBankRequest struct {
	Name                     *string `json:"name,omitempty"`
	ReflectMission           *string `json:"reflect_mission,omitempty"`
	RetainMission            *string `json:"retain_mission,omitempty"`
	RetainExtractionMode     *string `json:"retain_extraction_mode,omitempty"`
	RetainCustomInstructions *string `json:"retain_custom_instructions,omitempty"`
	RetainChunkSize          *int    `json:"retain_chunk_size,omitempty"`
	EnableObservations       *bool   `json:"enable_observations,omitempty"`
	ObservationsMission      *string `json:"observations_mission,omitempty"`
}

type RetainRequest struct {
	Items []MemoryItem `json:"items"`
	Async bool         `json:"async"`
}

type MemoryItem struct {
	Content           string            `json:"content"`
	Timestamp         *string           `json:"timestamp,omitempty"`
	Context           *string           `json:"context,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	DocumentID        *string           `json:"document_id,omitempty"`
	Tags              []string          `json:"tags,omitempty"`
	ObservationScopes any               `json:"observation_scopes,omitempty"`
	Strategy          *string           `json:"strategy,omitempty"`
	UpdateMode        *string           `json:"update_mode,omitempty"`
}

type RetainResponse struct {
	Success      bool        `json:"success"`
	BankID       string      `json:"bank_id"`
	ItemsCount   int         `json:"items_count"`
	Async        bool        `json:"async"`
	OperationID  *string     `json:"operation_id"`
	OperationIDs []string    `json:"operation_ids"`
	Usage        *TokenUsage `json:"usage"`
}

type RecallRequest struct {
	Query          string         `json:"query"`
	Types          []string       `json:"types,omitempty"`
	Budget         string         `json:"budget,omitempty"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	Trace          bool           `json:"trace"`
	QueryTimestamp *string        `json:"query_timestamp,omitempty"`
	Include        map[string]any `json:"include,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	TagsMatch      string         `json:"tags_match,omitempty"`
	TagGroups      []any          `json:"tag_groups,omitempty"`
}

type RecallResponse struct {
	Results     []RecallResult          `json:"results"`
	Trace       map[string]any          `json:"trace"`
	Entities    map[string]any          `json:"entities"`
	Chunks      map[string]any          `json:"chunks"`
	SourceFacts map[string]RecallResult `json:"source_facts"`
}

type RecallResult struct {
	ID            string            `json:"id"`
	Text          string            `json:"text"`
	Type          *string           `json:"type"`
	Entities      []string          `json:"entities"`
	Context       *string           `json:"context"`
	OccurredStart *string           `json:"occurred_start"`
	OccurredEnd   *string           `json:"occurred_end"`
	MentionedAt   *string           `json:"mentioned_at"`
	DocumentID    *string           `json:"document_id"`
	Metadata      map[string]string `json:"metadata"`
	ChunkID       *string           `json:"chunk_id"`
	Tags          []string          `json:"tags"`
	SourceFactIDs []string          `json:"source_fact_ids"`
}

type ReflectRequest struct {
	Query                 string         `json:"query"`
	Budget                string         `json:"budget,omitempty"`
	MaxTokens             int            `json:"max_tokens,omitempty"`
	Include               map[string]any `json:"include,omitempty"`
	ResponseSchema        map[string]any `json:"response_schema,omitempty"`
	Tags                  []string       `json:"tags,omitempty"`
	TagsMatch             string         `json:"tags_match,omitempty"`
	TagGroups             []any          `json:"tag_groups,omitempty"`
	FactTypes             []string       `json:"fact_types,omitempty"`
	ExcludeMentalModels   bool           `json:"exclude_mental_models"`
	ExcludeMentalModelIDs []string       `json:"exclude_mental_model_ids,omitempty"`
}

type ReflectResponse struct {
	Text             string         `json:"text"`
	BasedOn          map[string]any `json:"based_on"`
	StructuredOutput map[string]any `json:"structured_output"`
	Usage            *TokenUsage    `json:"usage"`
	Trace            map[string]any `json:"trace"`
}

type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
