package terminal

var gen_counter uint64 = 0

type Row struct {
	cells []Cell
	// gen is a generation counter.  It can be used to quickly rule
	// out the possibility of two rows being identical; this is useful
	// in scrolling.
	gen uint64
}

func getGen() uint64 {
	gen_counter += 1
	return gen_counter
}

func NewRow(width int, bgColor uint32) *Row {
	r := Row{}
	r.cells = make([]Cell, width)
	for i := range r.cells {
		r.cells[i].SetRenditions(Renditions{bgColor: bgColor})
		// fmt.Printf("NeRow: set cell %v %d\n", c.GetRenditions(), bgColor)
	}
	r.gen = getGen()
	// fmt.Printf("NewRow: %v\n", r.cells)
	return &r
}

func (r *Row) InsertCell(col int, bgColor uint32) bool {
	// validate the column range
	if col < 0 || col > len(r.cells)-1 {
		return false
	}

	// prepare the new cell
	cell := Cell{}
	cell.renditions = Renditions{bgColor: bgColor}

	// insert cell
	r.cells = append(r.cells[:col+1], r.cells[col:]...)
	r.cells[col] = cell

	// pop the last one
	width := len(r.cells) - 1
	r.cells = r.cells[:width]
	return true
}

func (r *Row) DeleteCell(col int, bgColor uint32) bool {
	if col < 0 || col > len(r.cells)-1 {
		return false
	}

	// prepare the new cell
	cell := Cell{}
	cell.renditions = Renditions{bgColor: bgColor}

	// add new cell at the end
	r.cells = append(r.cells, cell)

	// delete cell at col
	copy(r.cells[col:], r.cells[col+1:])

	// remvoe the last one
	width := len(r.cells) - 1
	r.cells = r.cells[:width]
	return true
}

func (r *Row) Reset(bgColor uint32) {
	r.gen = getGen()
	for i := range r.cells {
		r.cells[i].Reset(bgColor)
	}
}

func (r Row) GetWrap() bool {
	return r.cells[len(r.cells)-1].GetWrap()
}

func (r *Row) SetWrap(w bool) {
	r.cells[len(r.cells)-1].SetWrap(w)
}

func (r Row) Equal(other *Row) bool {

	// the easy way to compare
	if r.gen != other.gen {
		return false
	}

	// has different size?
	if len(r.cells) != len(other.cells) {
		return false
	}

	// check the content
	for i := range r.cells {
		if r.cells[i] != other.cells[i] {
			return false
		}
	}
	return true
}
