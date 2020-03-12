package plugin

type inProcessPlugin interface {
	Name() string
	Version() string
}

type InProcessCollector interface {
	Collector
	inProcessPlugin
}

type InProcessStreamingCollector interface {
	StreamingCollector
	inProcessPlugin
}

type InProcessPublisher interface {
	Publisher
	inProcessPlugin
}
