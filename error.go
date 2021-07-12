package lru

type LruErr struct {
	errMsg string
}

func (l LruErr) Error() string {
	return l.errMsg
}

var (
	ErrNotFound  = LruErr{"not found"}
	ErrValueType = LruErr{"value not is config node"}
)
