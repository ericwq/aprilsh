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

	// now, empty() should return false
	r.tl.x = 8
	r.tl.y = 9
	if r.empty() {
		t.Errorf("Rect.empty() should return %t, got %t\n", false, r.empty())
	}

	// after clear, empty() return true
	r.clear()
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

	// check rectangular first
	if r.rectangular {
		t.Errorf("First Rect.rectangular should return %t, got %t\n", false, r.rectangular)
	}

	// after toggle should return true
	r.toggleRectangular()
	got := r.rectangular
	if !got {
		t.Errorf("Second Rect.rectangular should return %t, got %t\n", true, got)
	}

	expectStr := "Rect{tl=(1,1) br=(9,9) rectangular=true}"
	gotStr := r.String()
	if gotStr != expectStr {
		t.Errorf("Rect.String expect %s, got %s\n", expectStr, gotStr)
	}
}

func TestDamage(t *testing.T) {
	tc := []struct {
		name         string
		start, end   int
		eStart, eEnd int
	}{
		{"1st round: start point", 6, 97, 6, 97}, // the last value pair is the base for the next round
		{"2nd round: extra left", 4, 90, 4, 97},
		{"3rd round: extra right", 96, 100, 4, 100},
		{"4th round: inside damage", 9, 90, 4, 100},
		{"5th round: equal start,end", 7, 7, 4, 100},
		{"reverse start,end", 98, 7, 0, 108},
	}
	// base condition: start=7, end=98, totalCells=108
	d := Damage{start: 7, end: 7, totalCells: 108}
	for _, v := range tc {
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
		// set the base for the next round
		d.add(v.eStart, v.eEnd)
	}
}

func TestPointAll(t *testing.T) {
	tc := []struct {
		name   string
		lhs    Point
		rhs    Point
		expect bool
	}{
		{"equal true", Point{1, 2}, Point{1, 2}, true},
		{"equal false", Point{1, 2}, Point{2, 2}, false},
		{"less true", Point{10, 2}, Point{20, 2}, true},
		{"less false", Point{10, 2}, Point{9, 2}, false},
		{"less equal true", Point{9, 2}, Point{9, 2}, true},
		{"less equal true", Point{8, 2}, Point{9, 2}, true},
		{"less equal false", Point{8, 3}, Point{9, 2}, false},
	}

	for i, v := range tc {
		lhs := v.lhs
		switch i {
		case 0, 1:
			if lhs.equal(v.rhs) != v.expect {
				t.Errorf("%s expect %t, got %t\n", v.name, v.expect, lhs.equal(v.rhs))
			}
		case 2, 3:
			if lhs.less(v.rhs) != v.expect {
				t.Errorf("%s expect %t, got %t\n", v.name, v.expect, lhs.less(v.rhs))
			}
		default:
			if lhs.lessEqual(v.rhs) != v.expect {
				t.Errorf("%s expect %t, got %t\n", v.name, v.expect, lhs.lessEqual(v.rhs))
			}
		}
	}
}

func TestLowerBound(t *testing.T) {
	type args struct {
		array  []int
		target int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "test0",
			args: args{array: []int{1, 10, 10, 10, 20, 30, 40, 50, 60}, target: 10},
			want: 1,
		},
		{
			name: "test1",
			args: args{array: []int{1, 10, 20, 30, 40, 50, 60}, target: 1},
			want: 0,
		},
		{
			name: "test2",
			args: args{array: []int{1, 10, 20, 30, 40, 50, 60}, target: 50},
			want: 5,
		},
		{
			name: "test3",
			args: args{array: []int{1, 10, 20, 30, 40, 50, 60}, target: 60},
			want: 6,
		},
		{
			name: "test4",
			args: args{array: []int{1, 10, 20, 30, 40, 50, 60}, target: 61},
			want: 7,
		},
		{
			name: "test5",
			args: args{array: []int{1, 10, 20, 30, 40, 50, 60}, target: 2},
			want: 1,
		},
		{
			name: "test6",
			args: args{array: []int{1, 10, 20, 30, 40, 50, 60}, target: 59},
			want: 6,
		},
		{
			name: "test7",
			args: args{array: []int{1, 10, 20, 30, 40, 50, 60, 60, 60}, target: 60},
			want: 6,
		},
		{
			name: "test8",
			args: args{array: []int{}, target: 1},
			want: 0,
		},
		{
			name: "test9",
			args: args{array: []int{-5, -2, -2, -2}, target: -4},
			want: 1,
		},
		{
			name: "test10",
			args: args{array: []int{1, 5, 5, 5, 5, 7, 7, 7, 7, 9}, target: 0},
			want: 0,
		},
		{
			name: "test11",
			args: args{array: []int{1, 2, 5, 8, 10}, target: 11},
			want: 5,
		},
		{
			name: "test12",
			args: args{array: []int{1, 2, 2, 2, 3, 3, 10}, target: 3},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LowerBound(tt.args.array, tt.args.target); got != tt.want {
				t.Errorf("LowerBound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tc := []struct {
		name          string
		value, expect int
	}{
		{"positive value", 23, 23},
		{"negative value", -23, 23},
		{"zero negative value", -0, 0},
		{"zero value", 0, 0},
	}

	for _, v := range tc {
		got := abs(v.value)
		if got != v.expect {
			t.Errorf("%s expect abs(%d)=%d, got %d\n", v.name, v.value, v.expect, got)
		}
	}
}
