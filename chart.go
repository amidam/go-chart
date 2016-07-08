package chart

import (
	"io"
	"math"

	"github.com/golang/freetype/truetype"
)

// Chart is what we're drawing.
type Chart struct {
	Title      string
	TitleStyle Style

	Width  int
	Height int

	Background      Style
	Canvas          Style
	Axes            Style
	FinalValueLabel Style

	XRange Range
	YRange Range

	Font   *truetype.Font
	Series []Series
}

// GetCanvasTop gets the top corner pixel.
func (c Chart) GetCanvasTop() int {
	return c.Canvas.Padding.GetTop(DefaultCanvasPadding.Top)
}

// GetCanvasLeft gets the left corner pixel.
func (c Chart) GetCanvasLeft() int {
	return c.Canvas.Padding.GetLeft(DefaultCanvasPadding.Left)
}

// GetCanvasBottom gets the bottom corner pixel.
func (c Chart) GetCanvasBottom() int {
	return c.Height - c.Canvas.Padding.GetBottom(DefaultCanvasPadding.Bottom)
}

// GetCanvasRight gets the right corner pixel.
func (c Chart) GetCanvasRight() int {
	return c.Width - c.Canvas.Padding.GetRight(DefaultCanvasPadding.Right)
}

// GetCanvasWidth returns the width of the canvas.
func (c Chart) GetCanvasWidth() int {
	return c.Width - (c.Canvas.Padding.GetLeft(DefaultCanvasPadding.Left) + c.Canvas.Padding.GetRight(DefaultCanvasPadding.Right))
}

// GetCanvasHeight returns the height of the canvas.
func (c Chart) GetCanvasHeight() int {
	return c.Height - (c.Canvas.Padding.GetTop(DefaultCanvasPadding.Top) + c.Canvas.Padding.GetBottom(DefaultCanvasPadding.Bottom))
}

// GetFont returns the text font.
func (c Chart) GetFont() (*truetype.Font, error) {
	if c.Font != nil {
		return c.Font, nil
	}
	return GetDefaultFont()
}

// Render renders the chart with the given renderer to the given io.Writer.
func (c *Chart) Render(provider RendererProvider, w io.Writer) error {
	xrange, yrange := c.initRanges()

	font, err := c.GetFont()
	if err != nil {
		return err
	}

	r := provider(c.Width, c.Height)
	r.SetFont(font)
	c.drawBackground(r)
	c.drawCanvas(r)
	c.drawAxes(r, xrange, yrange)
	for _, series := range c.Series {
		c.drawSeries(r, series, xrange, yrange)
	}
	c.drawTitle(r)
	return r.Save(w)
}

func (c Chart) initRanges() (xrange Range, yrange Range) {
	//iterate over each series, pull out the min/max for x,y
	var didSetFirstValues bool
	var globalMinY, globalMinX float64
	var globalMaxY, globalMaxX float64
	for _, s := range c.Series {
		seriesLength := s.Len()
		for index := 0; index < seriesLength; index++ {
			vx, vy := s.GetValue(index)
			if didSetFirstValues {
				if globalMinX > vx {
					globalMinX = vx
				}
				if globalMinY > vy {
					globalMinY = vy
				}
				if globalMaxX < vx {
					globalMaxX = vx
				}
				if globalMaxY < vy {
					globalMaxY = vy
				}
			} else {
				globalMinX, globalMaxX = vx, vx
				globalMinY, globalMaxY = vy, vy
				didSetFirstValues = true
			}
		}
	}

	if c.XRange.IsZero() {
		xrange.Min = globalMinX
		xrange.Max = globalMaxX
	} else {
		xrange.Min = c.XRange.Min
		xrange.Max = c.XRange.Max
	}
	xrange.Domain = c.GetCanvasWidth()

	if c.YRange.IsZero() {
		yrange.Min = globalMinY
		yrange.Max = globalMaxY
	} else {
		yrange.Min = c.YRange.Min
		yrange.Max = c.YRange.Max
	}
	yrange.Domain = c.GetCanvasHeight()

	return
}

func (c Chart) drawBackground(r Renderer) {
	r.SetFillColor(c.Background.GetFillColor(DefaultBackgroundColor))
	r.MoveTo(0, 0)
	r.LineTo(c.Width, 0)
	r.LineTo(c.Width, c.Height)
	r.LineTo(0, c.Height)
	r.LineTo(0, 0)
	r.Close()
	r.Fill()
}

func (c Chart) drawCanvas(r Renderer) {
	r.SetFillColor(c.Canvas.GetFillColor(DefaultCanvasColor))
	r.MoveTo(c.GetCanvasLeft(), c.GetCanvasTop())
	r.LineTo(c.GetCanvasRight(), c.GetCanvasTop())
	r.LineTo(c.GetCanvasRight(), c.GetCanvasBottom())
	r.LineTo(c.GetCanvasLeft(), c.GetCanvasBottom())
	r.LineTo(c.GetCanvasLeft(), c.GetCanvasTop())
	r.Fill()
	r.Close()
}

func (c Chart) drawAxes(r Renderer, xrange, yrange Range) {
	if c.Axes.Show {
		r.SetStrokeColor(c.Axes.GetStrokeColor(DefaultAxisColor))
		r.SetLineWidth(c.Axes.GetStrokeWidth(DefaultLineWidth))
		r.MoveTo(c.GetCanvasLeft(), c.GetCanvasBottom())
		r.LineTo(c.GetCanvasRight(), c.GetCanvasBottom())
		r.LineTo(c.GetCanvasRight(), c.GetCanvasTop())
		r.Stroke()

		c.drawAxesLabels(r, xrange, yrange)
	}
}

func (c Chart) drawAxesLabels(r Renderer, xrange, yrange Range) {

}

func (c Chart) drawSeries(r Renderer, s Series, xrange, yrange Range) {
	r.SetStrokeColor(s.GetStyle().GetStrokeColor(DefaultLineColor))
	r.SetLineWidth(s.GetStyle().GetStrokeWidth(DefaultLineWidth))

	if s.Len() == 0 {
		return
	}

	px := c.Canvas.Padding.GetLeft(DefaultCanvasPadding.Left)
	py := c.Canvas.Padding.GetTop(DefaultCanvasPadding.Top)

	cw := c.GetCanvasWidth()

	v0x, v0y := s.GetValue(0)
	x0 := cw - xrange.Translate(v0x)
	y0 := yrange.Translate(v0y)
	r.MoveTo(x0+px, y0+py)

	var vx, vy float64
	var x, y int
	for index := 1; index < s.Len(); index++ {
		vx, vy = s.GetValue(index)
		x = cw - xrange.Translate(vx)
		y = yrange.Translate(vy)
		r.LineTo(x+px, y+py)
	}
	r.Stroke()

	c.drawFinalValueLabel(r, s, yrange)
}

func (c Chart) drawFinalValueLabel(r Renderer, s Series, yrange Range) {
	if c.FinalValueLabel.Show {
		_, lv := s.GetValue(s.Len() - 1)
		_, ll := s.GetLabel(s.Len() - 1)

		py := c.Canvas.Padding.GetTop(DefaultCanvasPadding.Top)
		ly := yrange.Translate(lv) + py

		r.SetFontSize(c.FinalValueLabel.GetFontSize(DefaultFinalLabelFontSize))

		textWidth := r.MeasureText(ll)
		textHeight := int(math.Floor(DefaultFinalLabelFontSize))
		halfTextHeight := textHeight >> 1

		cx := c.GetCanvasRight() + int(c.Axes.GetStrokeWidth(DefaultAxisLineWidth))

		pt := c.FinalValueLabel.Padding.GetTop(DefaultFinalLabelPadding.Top)
		pl := c.FinalValueLabel.Padding.GetLeft(DefaultFinalLabelPadding.Left)
		pr := c.FinalValueLabel.Padding.GetRight(DefaultFinalLabelPadding.Right)
		pb := c.FinalValueLabel.Padding.GetBottom(DefaultFinalLabelPadding.Bottom)

		textX := cx + pl + DefaultFinalLabelDeltaWidth
		textY := ly + halfTextHeight

		ltlx := cx + pl + DefaultFinalLabelDeltaWidth
		ltly := ly - (pt + halfTextHeight)

		ltrx := cx + pl + pr + textWidth
		ltry := ly - (pt + halfTextHeight)

		lbrx := cx + pl + pr + textWidth
		lbry := ly + (pb + halfTextHeight)

		lblx := cx + DefaultFinalLabelDeltaWidth
		lbly := ly + (pb + halfTextHeight)

		//draw the shape...
		r.SetFillColor(c.FinalValueLabel.GetFillColor(DefaultFinalLabelBackgroundColor))
		r.SetStrokeColor(c.FinalValueLabel.GetStrokeColor(s.GetStyle().GetStrokeColor(DefaultLineColor)))
		r.SetLineWidth(c.FinalValueLabel.GetStrokeWidth(DefaultAxisLineWidth))
		r.MoveTo(cx, ly)
		r.LineTo(ltlx, ltly)
		r.LineTo(ltrx, ltry)
		r.LineTo(lbrx, lbry)
		r.LineTo(lblx, lbly)
		r.LineTo(cx, ly)
		r.Close()
		r.FillStroke()

		r.SetFontColor(c.FinalValueLabel.GetFontColor(DefaultTextColor))
		r.Text(ll, textX, textY)
	}
}

func (c Chart) drawTitle(r Renderer) error {
	if len(c.Title) > 0 && c.TitleStyle.Show {
		r.SetFontColor(c.Canvas.GetFontColor(DefaultTextColor))
		titleFontSize := c.Canvas.GetFontSize(DefaultTitleFontSize)
		r.SetFontSize(titleFontSize)
		textWidth := r.MeasureText(c.Title)
		titleX := (c.Width >> 1) - (textWidth >> 1)
		titleY := c.GetCanvasTop() + int(titleFontSize)
		r.Text(c.Title, titleX, titleY)
	}
	return nil
}
