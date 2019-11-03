package time

import (
	"encoding/json"
	"fmt"
	"time"
)

type JSON struct {
	Time time.Time
}

func (j JSON) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, j.Time.UTC().Format(time.RFC3339))), nil
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalFormat(time.RFC3339, data)
	j.Time = parsed
	return err
}

type JSONNano struct {
	Time time.Time
}

func (j JSONNano) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, j.Time.UTC().Format(time.RFC3339Nano))), nil
}

func (j *JSONNano) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalFormat(time.RFC3339Nano, data)
	j.Time = parsed
	return err
}

type JSONUnix struct {
	Time time.Time
}

func (j JSONUnix) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`%d`, j.Time.UTC().Unix())), nil
}

func (j *JSONUnix) UnmarshalJSON(data []byte) error {
	var unix int64
	if err := json.Unmarshal(data, &unix); err != nil {
		return err
	}
	j.Time = time.Unix(unix, 0)
	return nil
}

func unmarshalFormat(layout string, data []byte) (time.Time, error) {
	var t string
	if err := json.Unmarshal(data, &t); err != nil {
		return time.Time{}, err
	}

	if parsed, err := time.Parse(layout, t); err != nil {
		return time.Time{}, err
	} else {
		return parsed.UTC(), nil
	}

}
