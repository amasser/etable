// Copyright (c) 2019, The eTable Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package eplot

import (
	"math"

	"github.com/emer/etable/etensor"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// GenPlotXY generates an XY (lines, points) plot, setting GPlot variable
func (pl *Plot2D) GenPlotXY() {
	plt, _ := plot.New() // todo: not clear how to re-use, due to newtablexynames
	plt.Title.Text = pl.Params.Title
	plt.X.Label.Text = pl.XLabel()
	plt.Y.Label.Text = pl.YLabel()
	plt.BackgroundColor = nil

	// process xaxis first
	xi, xbreaks, err := pl.PlotXAxis(plt)
	if err != nil {
		return
	}
	xp := pl.Cols[xi]

	var firstXY *TableXY
	var strCols []*ColParams

	for _, cp := range pl.Cols {
		cp.UpdateVals()
		if !cp.On {
			continue
		}
		if cp.IsString {
			strCols = append(strCols, cp)
			continue
		}
		if cp.Range.FixMin {
			plt.Y.Min = math.Min(plt.Y.Min, cp.Range.Min)
		}
		if cp.Range.FixMax {
			plt.Y.Max = math.Max(plt.Y.Max, cp.Range.Max)
		}
	}

	if xbreaks != nil {
		stRow := 0
		for bi, edRow := range xbreaks {
			firstXY = nil
			for _, cp := range pl.Cols {
				if !cp.On || cp == xp {
					continue
				}
				if cp.IsString {
					continue
				}
				xy, _ := NewTableXYName(pl.Table, stRow, edRow, xi, xp.TensorIdx, cp.Col, cp.TensorIdx)
				if firstXY == nil {
					firstXY = xy
				}
				var pts *plotter.Scatter
				var lns *plotter.Line
				if pl.Params.Lines && pl.Params.Points {
					lns, pts, _ = plotter.NewLinePoints(xy)
				} else if pl.Params.Points {
					pts, _ = plotter.NewScatter(xy)
				} else {
					lns, _ = plotter.NewLine(xy)
				}
				if lns != nil {
					lns.LineStyle.Width = vg.Points(pl.Params.LineWidth)
					lns.LineStyle.Color = cp.Color
					plt.Add(lns)
					if bi == 0 {
						plt.Legend.Add(cp.Label(), lns)
					}
				}
				if pts != nil {
					pts.GlyphStyle.Color = cp.Color
					pts.GlyphStyle.Radius = vg.Points(pl.Params.PointSize)
					plt.Add(pts)
					if lns == nil && bi == 0 {
						plt.Legend.Add(cp.Label(), pts)
					}
				}
				if cp.ErrCol != "" {
					ec := pl.Table.ColIdx(cp.ErrCol)
					if ec >= 0 {
						xy.ErrCol = ec
						eb, _ := plotter.NewYErrorBars(xy)
						plt.Add(eb)
					}
				}
			}
			if firstXY != nil && len(strCols) > 0 {
				for _, cp := range strCols {
					xy, _ := NewTableXYName(pl.Table, stRow, edRow, xi, xp.TensorIdx, cp.Col, cp.TensorIdx)
					xy.LblCol = xy.YCol
					xy.YCol = firstXY.YCol
					lbls, _ := plotter.NewLabels(xy)
					plt.Add(lbls)
				}
			}
			stRow = edRow
		}
	} else {
		stRow := 0
		edRow := pl.Table.Rows
		for _, cp := range pl.Cols {
			if !cp.On || cp == xp {
				continue
			}
			if cp.IsString {
				continue
			}
			if cp.Range.FixMin {
				plt.Y.Min = math.Min(plt.Y.Min, cp.Range.Min)
			}
			if cp.Range.FixMax {
				plt.Y.Max = math.Max(plt.Y.Max, cp.Range.Max)
			}

			xy, _ := NewTableXYName(pl.Table, stRow, edRow, xi, xp.TensorIdx, cp.Col, cp.TensorIdx)
			if firstXY == nil {
				firstXY = xy
			}
			var pts *plotter.Scatter
			var lns *plotter.Line
			if pl.Params.Lines && pl.Params.Points {
				lns, pts, _ = plotter.NewLinePoints(xy)
			} else if pl.Params.Points {
				pts, _ = plotter.NewScatter(xy)
			} else {
				lns, _ = plotter.NewLine(xy)
			}
			if lns != nil {
				lns.LineStyle.Width = vg.Points(pl.Params.LineWidth)
				lns.LineStyle.Color = cp.Color
				plt.Add(lns)
				plt.Legend.Add(cp.Label(), lns)
			}
			if pts != nil {
				pts.GlyphStyle.Color = cp.Color
				pts.GlyphStyle.Radius = vg.Points(pl.Params.PointSize)
				plt.Add(pts)
				if lns == nil {
					plt.Legend.Add(cp.Label(), pts)
				}
			}
			if cp.ErrCol != "" {
				ec := pl.Table.ColIdx(cp.ErrCol)
				if ec >= 0 {
					xy.ErrCol = ec
					eb, _ := plotter.NewYErrorBars(xy)
					plt.Add(eb)
				}
			}
		}
		if firstXY != nil && len(strCols) > 0 {
			for _, cp := range strCols {
				xy, _ := NewTableXYName(pl.Table, stRow, edRow, xi, xp.TensorIdx, cp.Col, cp.TensorIdx)
				xy.LblCol = xy.YCol
				xy.YCol = firstXY.YCol
				lbls, _ := plotter.NewLabels(xy)
				plt.Add(lbls)
			}
		}
	}

	// Use string labels for X axis if X is a string
	xc := pl.Table.Cols[xi]
	if xc.DataType() == etensor.STRING {
		xcs := xc.(*etensor.String)
		plt.NominalX(xcs.Values...)
	}

	plt.Legend.Top = true
	pl.GPlot = plt
}