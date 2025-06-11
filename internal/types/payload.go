package types

// PayloadContext holds information extracted from the request payload
// that can be used for routing and model selection decisions
type PayloadContext struct {
	OriginalModel string
	HasStream     bool
	HasTools      bool
	HasImages     bool
	HasVideos     bool
	MessagesCount int
} 