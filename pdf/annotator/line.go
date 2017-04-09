package annotator

import (
	"math"

	"github.com/unidoc/unidoc/common"
	pdfcontent "github.com/unidoc/unidoc/pdf/contentstream"
	"github.com/unidoc/unidoc/pdf/contentstream/draw"
	pdfcore "github.com/unidoc/unidoc/pdf/core"
	pdf "github.com/unidoc/unidoc/pdf/model"
)

// Make the path with the content creator.
func drawPathWithCreator(path draw.Path, creator *pdfcontent.ContentCreator) {
	for idx, p := range path.Points {
		if idx == 0 {
			creator.Add_m(p.X, p.Y)
		} else {
			creator.Add_l(p.X, p.Y)
		}
	}
}

func makeLineAnnotationAppearanceStream(lineDef LineAnnotationDef) (*pdfcore.PdfObjectDictionary, *pdf.PdfRectangle, error) {
	form := pdf.NewXObjectForm()
	form.Resources = pdf.NewPdfPageResources()

	gsName := ""
	if lineDef.Opacity < 1.0 {
		// Create graphics state with right opacity.
		gsState := &pdfcore.PdfObjectDictionary{}
		gsState.Set("ca", pdfcore.MakeFloat(lineDef.Opacity))
		err := form.Resources.AddExtGState("gs1", gsState)
		if err != nil {
			common.Log.Debug("Unable to add extgstate gs1")
			return nil, nil, err
		}

		gsName = "gs1"
	}

	content, linebbox, err := drawPdfLine(lineDef, gsName)
	if err != nil {
		return nil, nil, err
	}

	err = form.SetContentStream(content, nil)
	if err != nil {
		return nil, nil, err
	}

	form.BBox = linebbox.ToPdfObject()

	apDict := &pdfcore.PdfObjectDictionary{}
	apDict.Set("N", form.ToPdfObject())

	return apDict, linebbox, nil
}

// Draw a line in PDF.  Generates the content stream which can be used in page contents or appearance stream of annotation.
func drawPdfLine(lineDef LineAnnotationDef, gsName string) ([]byte, *pdf.PdfRectangle, error) {
	x1, x2 := lineDef.X1, lineDef.X2
	y1, y2 := lineDef.Y1, lineDef.Y2

	dy := y2 - y1
	dx := x2 - x1
	theta := math.Atan2(dy, dx)

	L := math.Sqrt(math.Pow(dx, 2.0) + math.Pow(dy, 2.0))
	w := lineDef.LineWidth

	pi := math.Pi

	mul := 1.0
	if dx < 0 {
		mul *= -1.0
	}
	if dy < 0 {
		mul *= -1.0
	}

	// Vs.
	VsX := mul * (-w / 2 * math.Cos(theta+pi/2))
	VsY := mul * (-w/2*math.Sin(theta+pi/2) + w*math.Sin(theta+pi/2))

	// V1.
	V1X := VsX + w/2*math.Cos(theta+pi/2)
	V1Y := VsY + w/2*math.Sin(theta+pi/2)

	// P2.
	V2X := VsX + w/2*math.Cos(theta+pi/2) + L*math.Cos(theta)
	V2Y := VsY + w/2*math.Sin(theta+pi/2) + L*math.Sin(theta)

	// P3.
	V3X := VsX + w/2*math.Cos(theta+pi/2) + L*math.Cos(theta) + w*math.Cos(theta-pi/2)
	V3Y := VsY + w/2*math.Sin(theta+pi/2) + L*math.Sin(theta) + w*math.Sin(theta-pi/2)

	// P4.
	V4X := VsX + w/2*math.Cos(theta-pi/2)
	V4Y := VsY + w/2*math.Sin(theta-pi/2)

	path := draw.NewPath()
	path = path.AppendPoint(draw.NewPoint(V1X, V1Y))
	path = path.AppendPoint(draw.NewPoint(V2X, V2Y))
	path = path.AppendPoint(draw.NewPoint(V3X, V3Y))
	path = path.AppendPoint(draw.NewPoint(V4X, V4Y))

	lineEnding1 := lineDef.LineEndingStyle1
	lineEnding2 := lineDef.LineEndingStyle2

	// TODO: Allow custom height/widths.
	arrowHeight := 3 * w
	arrowWidth := 3 * w
	arrowExtruding := (arrowWidth - w) / 2

	if lineEnding2 == LineEndingStyleArrow {
		// Convert P2, P3
		p2 := path.GetPointNumber(2)

		va1 := draw.NewVectorPolar(arrowHeight, theta+pi)
		pa1 := p2.AddVector(va1)

		bVec := draw.NewVectorPolar(arrowWidth/2, theta+pi/2)
		aVec := draw.NewVectorPolar(arrowHeight, theta)

		va2 := draw.NewVectorPolar(arrowExtruding, theta+pi/2)
		pa2 := pa1.AddVector(va2)

		va3 := aVec.Add(bVec.Flip())
		pa3 := pa2.AddVector(va3)

		va4 := bVec.Scale(2).Flip().Add(va3.Flip())
		pa4 := pa3.AddVector(va4)

		pa5 := pa1.AddVector(draw.NewVectorPolar(w, theta-pi/2))

		newpath := draw.NewPath()
		newpath = newpath.AppendPoint(path.GetPointNumber(1))
		newpath = newpath.AppendPoint(pa1)
		newpath = newpath.AppendPoint(pa2)
		newpath = newpath.AppendPoint(pa3)
		newpath = newpath.AppendPoint(pa4)
		newpath = newpath.AppendPoint(pa5)
		newpath = newpath.AppendPoint(path.GetPointNumber(4))

		path = newpath
	}
	if lineEnding1 == LineEndingStyleArrow {
		// Get the first and last points.
		p1 := path.GetPointNumber(1)
		pn := path.GetPointNumber(path.Length())

		// First three points on arrow.
		v1 := draw.NewVectorPolar(w/2, theta+pi+pi/2)
		pa1 := p1.AddVector(v1)

		v2 := draw.NewVectorPolar(arrowHeight, theta).Add(draw.NewVectorPolar(arrowWidth/2, theta+pi/2))
		pa2 := pa1.AddVector(v2)

		v3 := draw.NewVectorPolar(arrowExtruding, theta-pi/2)
		pa3 := pa2.AddVector(v3)

		// Last three points
		v5 := draw.NewVectorPolar(arrowHeight, theta)
		pa5 := pn.AddVector(v5)

		v6 := draw.NewVectorPolar(arrowExtruding, theta+pi+pi/2)
		pa6 := pa5.AddVector(v6)

		pa7 := pa1

		newpath := draw.NewPath()
		newpath = newpath.AppendPoint(pa1)
		newpath = newpath.AppendPoint(pa2)
		newpath = newpath.AppendPoint(pa3)
		for _, p := range path.Points[1 : len(path.Points)-1] {
			newpath = newpath.AppendPoint(p)
		}
		newpath = newpath.AppendPoint(pa5)
		newpath = newpath.AppendPoint(pa6)
		newpath = newpath.AppendPoint(pa7)

		path = newpath
	}

	pathBbox := path.GetBoundingBox()
	creator := pdfcontent.NewContentCreator()

	// Draw line with arrow
	creator.
		Add_q().
		Add_rg(lineDef.LineColor.R(), lineDef.LineColor.G(), lineDef.LineColor.B())
	if len(gsName) > 1 {
		// If a graphics state is provided, use it. (Used for transparency settings here).
		creator.Add_gs(pdfcore.PdfObjectName(gsName))
	}
	drawPathWithCreator(path, creator)
	creator.Add_f().
		//creator.Add_S().
		Add_Q()

	bbox := &pdf.PdfRectangle{}
	bbox.Llx = pathBbox.X
	bbox.Lly = pathBbox.Y
	bbox.Urx = pathBbox.X + pathBbox.Width
	bbox.Ury = pathBbox.Y + pathBbox.Height

	return creator.Bytes(), bbox, nil
}