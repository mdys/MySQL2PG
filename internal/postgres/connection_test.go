package postgres

import (
	"database/sql"
	"testing"
)

func TestMakeTypedDestUsesNullableTypes(t *testing.T) {
	cases := []struct {
		name     string
		colType  string
		typeName string
	}{
		{name: "int", colType: "int(11)", typeName: "*sql.NullInt64"},
		{name: "decimal", colType: "decimal(10,2)", typeName: "*sql.NullString"},
		{name: "float", colType: "float", typeName: "*sql.NullFloat64"},
		{name: "bool", colType: "boolean", typeName: "*sql.NullBool"},
		{name: "varchar", colType: "varchar(64)", typeName: "*sql.NullString"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dest := makeTypedDest(tc.colType)
			switch tc.typeName {
			case "*sql.NullInt64":
				if _, ok := dest.value.(*sql.NullInt64); !ok {
					t.Fatalf("期望 *sql.NullInt64，实际 %T", dest.value)
				}
			case "*sql.NullString":
				if _, ok := dest.value.(*sql.NullString); !ok {
					t.Fatalf("期望 *sql.NullString，实际 %T", dest.value)
				}
			case "*sql.NullFloat64":
				if _, ok := dest.value.(*sql.NullFloat64); !ok {
					t.Fatalf("期望 *sql.NullFloat64，实际 %T", dest.value)
				}
			case "*sql.NullBool":
				if _, ok := dest.value.(*sql.NullBool); !ok {
					t.Fatalf("期望 *sql.NullBool，实际 %T", dest.value)
				}
			}
		})
	}
}

func TestGetTypedValueHandlesNullAndValidValues(t *testing.T) {
	intNull := typedDest{value: &sql.NullInt64{}}
	if got := getTypedValue(&intNull); got != nil {
		t.Fatalf("期望 nil，实际 %v", got)
	}

	intValue := typedDest{value: &sql.NullInt64{Int64: 42, Valid: true}}
	if got := getTypedValue(&intValue); got != int64(42) {
		t.Fatalf("期望 42，实际 %v", got)
	}

	strNull := typedDest{value: &sql.NullString{}}
	if got := getTypedValue(&strNull); got != nil {
		t.Fatalf("期望 nil，实际 %v", got)
	}

	strValue := typedDest{value: &sql.NullString{String: "ok", Valid: true}}
	if got := getTypedValue(&strValue); got != "ok" {
		t.Fatalf("期望 ok，实际 %v", got)
	}
}

func TestResetTypedDestinationsResetsNullableState(t *testing.T) {
	dests := []typedDest{
		{value: &sql.NullInt64{Int64: 9, Valid: true}},
		{value: &sql.NullString{String: "x", Valid: true}},
		{value: &sql.NullFloat64{Float64: 1.2, Valid: true}},
		{value: &sql.NullBool{Bool: true, Valid: true}},
	}

	resetTypedDestinations(dests)

	if v := dests[0].value.(*sql.NullInt64); v.Valid || v.Int64 != 0 {
		t.Fatalf("NullInt64 未正确重置: %+v", *v)
	}
	if v := dests[1].value.(*sql.NullString); v.Valid || v.String != "" {
		t.Fatalf("NullString 未正确重置: %+v", *v)
	}
	if v := dests[2].value.(*sql.NullFloat64); v.Valid || v.Float64 != 0 {
		t.Fatalf("NullFloat64 未正确重置: %+v", *v)
	}
	if v := dests[3].value.(*sql.NullBool); v.Valid || v.Bool {
		t.Fatalf("NullBool 未正确重置: %+v", *v)
	}
}
