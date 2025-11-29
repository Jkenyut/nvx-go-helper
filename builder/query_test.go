package builder

import (
	"reflect"
	"strings"
	"testing"
)

func assertStringEqual(t *testing.T, got, want string) {
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func assertStringsContain(t *testing.T, haystack, needle string) {
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected %q to contain %q", haystack, needle)
	}
}

func assertArgsEqual(t *testing.T, got, want []any) {
	if len(got) != len(want) {
		t.Errorf("len got %d want %d", len(got), len(want))
		return
	}
	for i := 0; i < len(got); i++ {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("at index %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestAllFeatures(t *testing.T) {
	sql, args := New().
		Eq("status", "active").
		NotEq("deleted", true).
		Gt("score", 100).
		Lte("age", 30).
		Like("name", "john").
		StartsWith("email", "admin").
		EndsWith("title", "CEO").
		IsNull("deleted_at").
		NotNull("email").
		Between("salary", 50000, 200000).
		In("role", "admin", "moderator").
		In("id", []int{1, 2, 3}).
		NotIn("status", []string{"banned"}).
		Group(func(g *WhereBuilder) {
			g.Eq("country", "ID")
			g.Lt("balance", 1000)
		}).
		OrGroup(func(g *WhereBuilder) {
			g.Eq("vip", true)
			g.Gte("joined_at", "2024-01-01")
		}).
		OrderBy("created_at", "desc").
		Random().
		Build()

	expectedSQL := `WHERE status = ? AND deleted != ? AND score > ? AND age <= ? AND name LIKE ? AND email LIKE ? AND title LIKE ? AND deleted_at IS NULL AND email IS NOT NULL AND salary BETWEEN ? AND ? AND role IN (?, ?) AND id IN (?, ?, ?) AND status NOT IN (?) AND (country = ? AND balance < ?) AND (vip = ? OR joined_at >= ?) ORDER BY created_at DESC, RANDOM()`

	expectedArgs := []any{
		"active", true, 100, 30, "%john%", "admin%", "%CEO",
		50000, 200000,
		"admin", "moderator",
		1, 2, 3,
		"banned",
		"ID", 1000,
		true, "2024-01-01",
	}

	assertStringEqual(t, sql, expectedSQL)
	assertArgsEqual(t, args, expectedArgs)
}
func TestEmptyInNotIn(t *testing.T) {
	sql, args := New().In("id", []int{}).Build()
	assertStringsContain(t, sql, "WHERE 1 = 0")
	if len(args) != 0 {
		t.Errorf("expected empty args, got %v", args)
	}

	sql, args = New().NotIn("status", []string{}).Build()
	assertStringsContain(t, sql, "WHERE 1 = 1")
	if len(args) != 0 {
		t.Errorf("expected empty args, got %v", args)
	}
}

func TestOrGroupAsFirstCondition(t *testing.T) {
	sql, _ := New().
		OrGroup(func(g *WhereBuilder) {
			g.Eq("a", 1)
			g.Eq("b", 2)
		}).
		Eq("c", 3).
		Build()

	assertStringsContain(t, sql, "WHERE (a = ? OR b = ?) AND c = ?")
}

func TestNestedGroups(t *testing.T) {
	sql, _ := New().
		Group(func(g *WhereBuilder) {
			g.Eq("x", 1).
				OrGroup(func(og *WhereBuilder) {
					og.Eq("y", 2)
					og.Eq("z", 3)
				})
		}).
		Build()

	assertStringsContain(t, sql, "WHERE (x = ? AND (y = ? OR z = ?))")
}

func TestOrderByMulti(t *testing.T) {
	sql, _ := New().
		OrderByMulti("name ASC", "age", "score DESC").
		Build()

	assertStringsContain(t, sql, "ORDER BY name ASC, age ASC, score DESC")
}

func TestRandom(t *testing.T) {
	sql, _ := New().Random().Build()
	assertStringsContain(t, sql, "ORDER BY RANDOM()")
}

func TestNoConditions(t *testing.T) {
	sql, args := New().Build()
	assertStringEqual(t, sql, "")
	if len(args) != 0 {
		t.Errorf("expected empty args, got %v", args)
	}
}

func TestReset(t *testing.T) {
	b := New().Eq("a", 1).OrderBy("b", "desc")
	b.Reset()
	sql, args := b.Build()
	assertStringEqual(t, sql, "")
	if len(args) != 0 {
		t.Errorf("expected empty args, got %v", args)
	}
}

func TestInWithVariousTypes(t *testing.T) {
	ids := []int64{5, 6, 7}
	roles := []string{"user", "guest"}

	sql, args := New().
		In("id", ids).
		In("role", roles).
		Build()

	assertStringsContain(t, sql, "id IN (?, ?, ?)")
	assertStringsContain(t, sql, "role IN (?, ?)")
	expectedArgs := []any{int64(5), int64(6), int64(7), "user", "guest"}
	assertArgsEqual(t, args, expectedArgs)
}

func TestRawAndWhere(t *testing.T) {
	sql, args := New().
		Raw("status = 'pending'").
		Where("created_at > ?", "2025-01-01").
		Build()

	assertStringEqual(t, sql, "WHERE status = 'pending' AND created_at > ?")
	assertArgsEqual(t, args, []any{"2025-01-01"})
}

func TestMultipleOrGroups(t *testing.T) {
	sql, _ := New().
		Eq("base", 1).
		OrGroup(func(g *WhereBuilder) {
			g.Eq("opt1", "a")
		}).
		OrGroup(func(g *WhereBuilder) {
			g.Eq("opt2", "b")
		}).
		Build()

	assertStringsContain(t, sql, "base = ? AND (opt1 = ?) AND (opt2 = ?)")
}

func TestEmptyGroup(t *testing.T) {
	sql, args := New().
		Group(func(g *WhereBuilder) {}).
		Eq("test", 1).
		Build()

	assertStringsContain(t, sql, "WHERE test = ?")
	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
}

func TestRawWithNoArgs(t *testing.T) {
	sql, args := New().Raw("1 = 1").Build()
	assertStringEqual(t, sql, "WHERE 1 = 1")
	assertArgsEqual(t, args, []any{})
}

func TestToSliceCoverage(t *testing.T) {
	// Case 1: v == nil
	if got := toSlice(nil); got != nil {
		t.Errorf("toSlice(nil) = %v, want nil", got)
	}

	// Case 2: v bukan slice/array (misal string, int, struct, dll)
	if got := toSlice("not a slice"); got != nil {
		t.Errorf("toSlice(string) = %v, want nil", got)
	}
	if got := toSlice(42); got != nil {
		t.Errorf("toSlice(int) = %v, want nil", got)
	}
	if got := toSlice(struct{ X int }{100}); got != nil {
		t.Errorf("toSlice(struct) = %v, want nil", got)
	}

	// Case 3: sudah tercover oleh TestInWithVariousTypes, tapi kita pastikan lagi
	slice := []int64{1, 2, 3}
	result := toSlice(slice)
	if result == nil || len(result) != 3 || result[0].(int64) != 1 {
		t.Errorf("toSlice failed on []int64: %v", result)
	}
}

func TestNotIn_EmptyCases(t *testing.T) {
	// Case 1: NotIn dipanggil tanpa argumen sama sekali
	sql1, args1 := New().NotIn("status").Build()
	if !strings.Contains(sql1, "WHERE 1 = 1") || len(args1) != 0 {
		t.Errorf("NotIn() tanpa argumen → got %q, %v", sql1, args1)
	}

	// Case 2: NotIn dengan satu argumen yang slice kosong
	emptySlice := []string{}
	sql2, args2 := New().NotIn("status", emptySlice).Build()
	if !strings.Contains(sql2, "WHERE 1 = 1") || len(args2) != 0 {
		t.Errorf("NotIn dengan slice kosong → got %q, %v", sql2, args2)
	}

	// Bonus: pastikan slice tidak kosong tetap bekerja normal (sudah dicover di test lain)
	sql3, args3 := New().NotIn("status", []string{"banned", "deleted"}).Build()
	if !strings.Contains(sql3, "status NOT IN (?, ?)") || len(args3) != 2 {
		t.Errorf("NotIn dengan slice tidak kosong gagal → got %q, %v", sql3, args3)
	}
}
