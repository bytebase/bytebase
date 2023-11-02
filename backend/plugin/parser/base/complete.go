package base

// CandidateType is the type of candidate.
type CandidateType string

// Candidate is the candidate for auto-completion.
type Candidate struct {
	Text string
	Type CandidateType
}
