// Copyright (c) 2019, The eTable Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package etable

import (
	"math"
	"sort"

	"github.com/emer/etable/etensor"
)

// LessFunc is a function used for sort comparisons that returns
// true if Table row i is less than Table row j -- these are the
// raw row numbers, which have already been projected through
// indexes when used for sorting via Idxs.
type LessFunc func(et *Table, i, j int) bool

// FilterFunc is a function used for filtering that returns
// true if Table row should be included in the current filtered
// view of the table, and false if it should be removed.
type FilterFunc func(et *Table, row int) bool

// IdxTable is an indexed wrapper around an etable.Table that provides a
// specific view onto the Table defined by the set of indexes.
// This provides an efficient way of sorting and filtering a table by only
// updating the indexes while doing nothing to the Table itself.
// To produce a table that has data actually organized according to the
// indexed order, call the NewTable method.
// IdxTable views on a table can also be organized together as Splits
// of the table rows, e.g., by grouping values along a given column.
type IdxTable struct {
	Table    *Table   `desc:"Table that we are an indexed view onto"`
	Idxs     []int    `desc:"current indexes into Table"`
	lessFunc LessFunc `copy:"-" view:"-" xml:"-" json:"-" desc:"current Less function used in sorting"`
}

// NewIdxTable returns a new IdxTable based on given table, initialized with sequential idxes
func NewIdxTable(et *Table) *IdxTable {
	ix := &IdxTable{}
	ix.SetTable(et)
	return ix
}

// SetTable sets as indexes into given table with sequential initial indexes
func (ix *IdxTable) SetTable(et *Table) {
	ix.Table = et
	ix.Sequential()
}

// Sequential sets indexes to sequential row-wise indexes into table
func (ix *IdxTable) Sequential() {
	if ix.Table == nil {
		ix.Idxs = nil
		return
	}
	ix.Idxs = make([]int, ix.Table.Rows)
	for i := range ix.Idxs {
		ix.Idxs[i] = i
	}
}

// AddIndex adds a new index to the list
func (ix *IdxTable) AddIndex(idx int) {
	ix.Idxs = append(ix.Idxs, idx)
}

// Sort sorts the indexes into our Table using given Less function.
// The Less function operates directly on row numbers into the Table
// as these row numbers have already been projected through the indexes.
func (ix *IdxTable) Sort(lessFunc LessFunc) {
	ix.lessFunc = lessFunc
	sort.Sort(ix)
}

// SortCol sorts the indexes into our Table according to values in
// given column index, using either ascending or descending order.
// Only valid for 1-dimensional columns.
func (ix *IdxTable) SortCol(colIdx int, ascending bool) {
	cl := ix.Table.Cols[colIdx]
	if cl.DataType() == etensor.STRING {
		ix.Sort(func(et *Table, i, j int) bool {
			if ascending {
				return cl.StringVal1D(i) < cl.StringVal1D(j)
			} else {
				return cl.StringVal1D(i) > cl.StringVal1D(j)
			}
		})
	} else {
		ix.Sort(func(et *Table, i, j int) bool {
			if ascending {
				return cl.FloatVal1D(i) < cl.FloatVal1D(j)
			} else {
				return cl.FloatVal1D(i) > cl.FloatVal1D(j)
			}
		})
	}
}

// SortCols sorts the indexes into our Table according to values in
// given list of column indexes, using either ascending or descending order for
// all of the columns.  Only valid for 1-dimensional columns.
func (ix *IdxTable) SortCols(colIdxs []int, ascending bool) {
	ix.Sort(func(et *Table, i, j int) bool {
		for _, ci := range colIdxs {
			cl := ix.Table.Cols[ci]
			if cl.DataType() == etensor.STRING {
				if ascending {
					if cl.StringVal1D(i) < cl.StringVal1D(j) {
						return true
					} else if cl.StringVal1D(i) > cl.StringVal1D(j) {
						return false
					} // if equal, fallthrough to next col
				} else {
					if cl.StringVal1D(i) > cl.StringVal1D(j) {
						return true
					} else if cl.StringVal1D(i) < cl.StringVal1D(j) {
						return false
					} // if equal, fallthrough to next col
				}
			} else {
				if ascending {
					if cl.FloatVal1D(i) < cl.FloatVal1D(j) {
						return true
					} else if cl.FloatVal1D(i) < cl.FloatVal1D(j) {
						return false
					} // if equal, fallthrough to next col
				} else {
					if cl.FloatVal1D(i) > cl.FloatVal1D(j) {
						return true
					} else if cl.FloatVal1D(i) < cl.FloatVal1D(j) {
						return false
					} // if equal, fallthrough to next col
				}
			}
		}
		return false
	})
}

// Filter filters the indexes into our Table using given Filter function.
// The Filter function operates directly on row numbers into the Table
// as these row numbers have already been projected through the indexes.
func (ix *IdxTable) Filter(filterFunc FilterFunc) {
	sz := len(ix.Idxs)
	for i := sz - 1; i >= 0; i-- { // always go in reverse for filtering
		if !filterFunc(ix.Table, ix.Idxs[i]) { // delete
			ix.Idxs = append(ix.Idxs[:i], ix.Idxs[i+1:]...)
		}
	}
}

// NewTable returns a new table with column data organized according to
// the indexes
func (ix *IdxTable) NewTable() *Table {
	rows := len(ix.Idxs)
	sc := ix.Table.Schema()
	nt := New(sc, rows)
	if rows == 0 {
		return nt
	}
	for ci := range nt.Cols {
		scl := ix.Table.Cols[ci]
		tcl := nt.Cols[ci]
		_, csz := tcl.RowCellSize()
		for i, srw := range ix.Idxs {
			tcl.CopyCellsFrom(scl, i*csz, srw*csz, csz)
		}
	}
	return nt
}

// AggCol applies given aggregation function to each element in the given column, using float64
// conversions of the values.  init is the initial value for the agg variable.
// Operates independently over each cell on n-dimensional columns and returns the result as a slice
// of values per cell.
func (ix *IdxTable) AggCol(colIdx int, ini float64, fun etensor.AggFunc) []float64 {
	cl := ix.Table.Cols[colIdx]
	_, csz := cl.RowCellSize()

	ag := make([]float64, csz)
	for i := range ag {
		ag[i] = ini
	}
	if csz == 1 {
		for _, srw := range ix.Idxs {
			val := cl.FloatVal1D(srw)
			if !cl.IsNull1D(srw) && !math.IsNaN(val) {
				ag[0] = fun(srw, val, ag[0])
			}
		}
	} else {
		for _, srw := range ix.Idxs {
			si := srw * csz
			for j := range ag {
				val := cl.FloatVal1D(si + j)
				if !cl.IsNull1D(si+j) && !math.IsNaN(val) {
					ag[j] = fun(si+j, val, ag[j])
				}
			}
		}
	}
	return ag
}

// Clone returns a copy of the current index view with its own index memory
func (ix *IdxTable) Clone() *IdxTable {
	nix := &IdxTable{}
	nix.CopyFrom(ix)
	return nix
}

// CopyFrom copies from given other IdxTable (we have our own unique copy of indexes)
func (ix *IdxTable) CopyFrom(oix *IdxTable) {
	ix.Table = oix.Table
	ix.Idxs = make([]int, len(oix.Idxs))
	copy(ix.Idxs, oix.Idxs)
}

// Len returns the length of the index list
func (ix *IdxTable) Len() int {
	return len(ix.Idxs)
}

// Less calls the LessFunc for sorting
func (ix *IdxTable) Less(i, j int) bool {
	return ix.lessFunc(ix.Table, ix.Idxs[i], ix.Idxs[j])
}

// Swap switches the indexes for i and j
func (ix *IdxTable) Swap(i, j int) {
	ix.Idxs[i], ix.Idxs[j] = ix.Idxs[j], ix.Idxs[i]
}