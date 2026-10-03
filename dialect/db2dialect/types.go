package db2dialect

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"
)

// SmallIntBool represents a DB2 SMALLINT column used for boolean values (0 or 1).
// Older DB2 versions have no usable BOOLEAN column type, so booleans are stored
// as SMALLINT.
//
//	type Employee struct {
//	    Active db2dialect.SmallIntBool `bun:",default:1"`
//	}
type SmallIntBool int32

func (s *SmallIntBool) Scan(val interface{}) error {
	if val == nil {
		*s = 0
		return nil
	}

	switch v := val.(type) {
	case int32:
		*s = SmallIntBool(v)
	case int64:
		*s = SmallIntBool(v)
	case int:
		*s = SmallIntBool(v)
	case float64:
		*s = SmallIntBool(int32(v))
	case string:
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			switch v {
			case "true", "True", "TRUE", "yes", "Yes", "YES":
				*s = 1
			case "false", "False", "FALSE", "no", "No", "NO":
				*s = 0
			default:
				return fmt.Errorf("cannot scan %q into SmallIntBool", v)
			}
		} else {
			*s = SmallIntBool(n)
		}
	case []byte:
		return s.Scan(string(v))
	default:
		return fmt.Errorf("cannot scan %T into SmallIntBool", val)
	}
	return nil
}

func (s SmallIntBool) Value() (driver.Value, error) {
	return int64(s), nil
}

// Bool converts SmallIntBool to native Go bool.
func (s SmallIntBool) Bool() bool {
	return s != 0
}

// NullSmallIntBool represents a nullable DB2 SMALLINT boolean value.
type NullSmallIntBool struct {
	SmallIntBool SmallIntBool
	Valid        bool
}

func (ns *NullSmallIntBool) Scan(val interface{}) error {
	if val == nil {
		ns.SmallIntBool = 0
		ns.Valid = false
		return nil
	}
	var scanned SmallIntBool
	if err := scanned.Scan(val); err != nil {
		return err
	}
	ns.SmallIntBool = scanned
	ns.Valid = true
	return nil
}

func (ns NullSmallIntBool) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.SmallIntBool.Value()
}

// Bool converts NullSmallIntBool to *bool, returning nil when not valid.
func (ns NullSmallIntBool) Bool() *bool {
	if !ns.Valid {
		return nil
	}
	b := ns.SmallIntBool.Bool()
	return &b
}

// SmallInt represents a DB2 SMALLINT column as int32.
type SmallInt int32

func (si *SmallInt) Scan(val interface{}) error {
	if val == nil {
		*si = 0
		return nil
	}

	switch v := val.(type) {
	case int32:
		*si = SmallInt(v)
	case int64:
		*si = SmallInt(v)
	case int:
		*si = SmallInt(v)
	case float64:
		*si = SmallInt(int32(v))
	case string:
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return fmt.Errorf("cannot scan %q into SmallInt: %w", v, err)
		}
		*si = SmallInt(n)
	case []byte:
		return si.Scan(string(v))
	default:
		return fmt.Errorf("cannot scan %T into SmallInt", val)
	}
	return nil
}

func (si SmallInt) Value() (driver.Value, error) {
	return int64(si), nil
}

// NullSmallInt represents a nullable DB2 SMALLINT value.
type NullSmallInt struct {
	SmallInt SmallInt
	Valid    bool
}

func (ns *NullSmallInt) Scan(val interface{}) error {
	if val == nil {
		ns.SmallInt = 0
		ns.Valid = false
		return nil
	}
	var scanned SmallInt
	if err := scanned.Scan(val); err != nil {
		return err
	}
	ns.SmallInt = scanned
	ns.Valid = true
	return nil
}

func (ns NullSmallInt) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.SmallInt.Value()
}

// Int32 converts NullSmallInt to *int32, returning nil when not valid.
func (ns NullSmallInt) Int32() *int32 {
	if !ns.Valid {
		return nil
	}
	v := int32(ns.SmallInt)
	return &v
}

// SQLNullInt32 is an alias for sql.NullInt32, provided for consistency with
// the types above.
type SQLNullInt32 = sql.NullInt32

// Date represents a DB2 DATE column (no time-of-day component).
// The DB2 CLI driver binds plain time.Time values as a full TIMESTAMP
// structure, which DB2 rejects (SQL0180N) for DATE columns, so Date sends a
// "YYYY-MM-DD" string instead.
//
//	type Event struct {
//	    EventDate db2dialect.Date `bun:"type:DATE"`
//	}
type Date time.Time

func (d *Date) Scan(val interface{}) error {
	if val == nil {
		*d = Date(time.Time{})
		return nil
	}
	switch v := val.(type) {
	case time.Time:
		*d = Date(v)
	case string:
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return fmt.Errorf("cannot scan %q into Date: %w", v, err)
		}
		*d = Date(t)
	case []byte:
		return d.Scan(string(v))
	default:
		return fmt.Errorf("cannot scan %T into Date", val)
	}
	return nil
}

func (d Date) Value() (driver.Value, error) {
	return time.Time(d).Format("2006-01-02"), nil
}

// Time converts Date to native time.Time.
func (d Date) Time() time.Time {
	return time.Time(d)
}

// TimeOfDay represents a DB2 TIME column (no date component). Like Date, it is
// sent as a string ("HH:MM:SS") to avoid SQL0180N.
//
//	type Schedule struct {
//	    StartTime db2dialect.TimeOfDay `bun:"type:TIME"`
//	}
type TimeOfDay time.Time

func (t *TimeOfDay) Scan(val interface{}) error {
	if val == nil {
		*t = TimeOfDay(time.Time{})
		return nil
	}
	switch v := val.(type) {
	case time.Time:
		*t = TimeOfDay(v)
	case string:
		parsed, err := time.Parse("15:04:05", v)
		if err != nil {
			return fmt.Errorf("cannot scan %q into TimeOfDay: %w", v, err)
		}
		*t = TimeOfDay(parsed)
	case []byte:
		return t.Scan(string(v))
	default:
		return fmt.Errorf("cannot scan %T into TimeOfDay", val)
	}
	return nil
}

func (t TimeOfDay) Value() (driver.Value, error) {
	return time.Time(t).Format("15:04:05"), nil
}

// Time converts TimeOfDay to native time.Time.
func (t TimeOfDay) Time() time.Time {
	return time.Time(t)
}

// Timestamp represents a DB2 TIMESTAMP column. The DB2 CLI driver binds
// time.Time via SQL_TIMESTAMP_STRUCT, which can fail with SQL0180N depending on
// the column's fractional-seconds precision, so Timestamp sends a
// "YYYY-MM-DD HH:MM:SS.ffffff" string instead.
//
//	type Project struct {
//	    StartDate db2dialect.Timestamp `bun:""`
//	}
type Timestamp time.Time

func (ts *Timestamp) Scan(val interface{}) error {
	if val == nil {
		*ts = Timestamp(time.Time{})
		return nil
	}
	switch v := val.(type) {
	case time.Time:
		*ts = Timestamp(v)
	case string:
		t, err := time.Parse("2006-01-02 15:04:05.000000", v)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05", v)
			if err != nil {
				return fmt.Errorf("cannot scan %q into Timestamp: %w", v, err)
			}
		}
		*ts = Timestamp(t)
	case []byte:
		return ts.Scan(string(v))
	default:
		return fmt.Errorf("cannot scan %T into Timestamp", val)
	}
	return nil
}

func (ts Timestamp) Value() (driver.Value, error) {
	return time.Time(ts).Format("2006-01-02 15:04:05.000000"), nil
}

// Time converts Timestamp to native time.Time.
func (ts Timestamp) Time() time.Time {
	return time.Time(ts)
}
