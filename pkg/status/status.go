package status

const (
	OK            = 200
	Incomplete    = 206
	ParseError    = 400
	NeedsSecret   = 401
	UnknownFormat = 415
	RateLimited   = 429
	Unavailable   = 503
)

func Class(code int) int {
	if code < 100 {
		return 0
	}
	return code / 100
}
