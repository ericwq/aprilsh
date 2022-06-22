package terminal

import "testing"

func TestMinMax(t *testing.T) {
	tc := []struct {
		name   string
		min    bool
		values [3]int
	}{
		{"normal min 1", true, [3]int{44, 34, 34}},
		{"normal min 2", true, [3]int{44, 54, 44}},
		{"normal max", false, [3]int{44, 34, 44}},
		{"equal min", true, [3]int{34, 34, 34}},
		{"equal max", false, [3]int{34, 34, 34}},
	}

	for _, v := range tc {
		if v.min {
			got := min(v.values[0], v.values[1])
			expect := v.values[2]
			if got != expect {
				t.Errorf("%s expect min(%d,%d)=%d, got %d\n", v.name, v.values[0], v.values[1], expect, got)
			}
		} else {
			got := max(v.values[0], v.values[1])
			expect := v.values[2]
			if got != expect {
				t.Errorf("%s expect max(%d,%d)=%d, got %d\n", v.name, v.values[0], v.values[1], expect, got)
			}
		}
	}
}

func TestRectAll(t *testing.T) {
	r := NewRect()
	// a blank Rect.nul() return true
	if !r.null() {
		t.Errorf("Rect.null() should return %t, got %t\n", true, r.null())
	}
	r.tl.x = 8
	r.tl.y = 9

	// now, empty() should return false
	if r.empty() {
		t.Errorf("Rect.empty() should return %t, got %t\n", false, r.empty())
	}

	r.clear()

	// after clear, empty() return true
	if !r.empty() {
		t.Errorf("Rect.empty() should return %t, got %t\n", true, r.empty())
	}

	// prepare for the Rect.mid()
	r.tl.x = 1
	r.tl.y = 1
	r.br.x = 9
	r.br.y = 9
	expect := Point{5, 5}
	if expect != r.mid() {
		t.Errorf("Rect.mid() expect %v, got %v\n", expect, r.mid())
	}

	if r.rectangular {
		t.Errorf("First Rect.rectangular should return %t, got %t\n", false, r.rectangular)
	}

	// toggle should return true
	r.toggleRectangular()
	got := r.rectangular
	if !got {
		t.Errorf("Second Rect.rectangular should return %t, got %t\n", true, got)
	}
}

func TestDamage(t *testing.T) {
	tc := []struct {
		name         string
		start, end   int
		eStart, eEnd int
	}{
		{"extra left", 1, 9, 1, 98},
		{"extra right", 96, 100, 7, 100},
		{"inside damage", 9, 90, 7, 98},
		{"equal start,end", 7, 7, 7, 7},
		{"reverse start,end", 98, 7, 0, 108},
	}
	for _, v := range tc {
		// base condition: start=7, end=98, totalCells=108
		d := Damage{start: 7, end: 98, totalCells: 108}
		d.add(v.start, v.end)
		gotStart := d.start
		gotEnd := d.end

		if gotStart != v.eStart || gotEnd != v.eEnd {
			t.Errorf("%s expect (%d,%d), got (%d,%d)\n", v.name, v.eStart, v.eEnd, gotStart, gotEnd)
		}

		d.expose()
		if d.start != 0 || d.end != d.totalCells {
			t.Errorf("%s expect (%d,%d), got (%d,%d)\n", v.name, 0, 108, d.start, d.end)
		}

		d.reset()
		if d.start != 0 || d.end != 0 {
			t.Errorf("%s expect (%d,%d), got (%d,%d)\n", v.name, 0, 0, d.start, d.end)
		}
	}
}
