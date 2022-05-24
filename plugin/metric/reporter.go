package metric

// Reporter is the API message for metric reporter.
type Reporter interface {
	Close()
	Report(metric Metric) error
	Identify(identifier *Identifier) error
}
