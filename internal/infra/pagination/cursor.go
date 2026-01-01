package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("invalid cursor")

func Encode(createdAt time.Time, id uuid.UUID) string {
	raw := fmt.Sprintf("%d|%s", createdAt.UTC().UnixNano(), id.String())
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func Decode(cursor string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.UUID{}, ErrInvalidCursor
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.UUID{}, ErrInvalidCursor
	}
	nanos, err := parseInt64(parts[0])
	if err != nil {
		return time.Time{}, uuid.UUID{}, ErrInvalidCursor
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.UUID{}, ErrInvalidCursor
	}
	return time.Unix(0, nanos).UTC(), id, nil
}

func parseInt64(value string) (int64, error) {
	if value == "" {
		return 0, errors.New("empty")
	}
	var out int64
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch < '0' || ch > '9' {
			return 0, errors.New("invalid")
		}
		out = out*10 + int64(ch-'0')
	}
	return out, nil
}
