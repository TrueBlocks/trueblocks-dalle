// Command foldvis renders the deterministic 3-D attribute-selection chain
// described in dalle/design/deterministic-cartography-of-ai-mind-space.md.
//
// Usage:
//
//	go run ./cmd/foldvis -o figure.png
//	go run ./cmd/foldvis -o figure.png -seed "my seed phrase" -series "my series" -n 32
//
// With no -seed, the tool uses synthetic deterministic seed chunks so the
// output matches the illustrative figure in the design note.
package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	width  = 720
	height = 440
	super  = 4 // oversampling factor for anti-aliasing
)

type Vec3 struct{ X, Y, Z float64 }

func (v Vec3) Add(u Vec3) Vec3 { return Vec3{v.X + u.X, v.Y + u.Y, v.Z + u.Z} }
func (v Vec3) Mul(s float64) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3) Len() float64 { return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z) }
func (v Vec3) Normalize() Vec3 {
	l := v.Len()
	if l == 0 {
		return v
	}
	return v.Mul(1 / l)
}

func project(v Vec3, d float64) Vec3 {
	denom := d + v.Z
	return Vec3{v.X / denom, v.Y / denom, d / denom}
}

func main() {
	var (
		outPath = flag.String("o", "", "output PNG path (required)")
		seed    = flag.String("seed", "", "optional seed phrase; empty uses synthetic deterministic chunks")
		series  = flag.String("series", "", "optional series phrase")
		nBonds  = flag.Int("n", 24, "number of bonds")
	)
	flag.Parse()

	if *outPath == "" {
		fmt.Fprintln(os.Stderr, "usage: foldvis -o figure.png")
		flag.PrintDefaults()
		os.Exit(2)
	}

	img, err := renderFold(*seed, *series, *nBonds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}

	f, err := os.Create(*outPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
	fmt.Println("created", *outPath)
}

func renderFold(seed, series string, n int) (*image.RGBA, error) {
	// Load font.
	face, err := loadFont(14 * super)
	if err != nil {
		return nil, err
	}
	smallFace, err := loadFont(10 * super)
	if err != nil {
		return nil, err
	}

	// Render at super-resolution, then downsample.
	sw, sh := width*super, height*super
	src := image.NewRGBA(image.Rect(0, 0, sw, sh))
	draw.Draw(src, src.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	// Title
	label(src, face, 20*super, 25*super, "Figure 2: Each normalized attribute becomes one bond of a 3-D chain", color.Black)

	// Build chain.
	points := make([]Vec3, n+1)
	points[0] = Vec3{0, 0, 0}
	for i := 1; i <= n; i++ {
		chunk := seedChunk(seed, series, i)
		u, theta, phi := chunkToBond(chunk)
		dir := Vec3{
			math.Sin(phi) * math.Cos(theta),
			math.Sin(phi) * math.Sin(theta),
			math.Cos(phi),
		}.Normalize()
		points[i] = points[i-1].Add(dir.Mul(u))
	}

	// Project and find bounding box.
	const camD = 5.0
	proj := make([]Vec3, len(points))
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for i, p := range points {
		proj[i] = project(p, camD)
		if proj[i].X < minX {
			minX = proj[i].X
		}
		if proj[i].X > maxX {
			maxX = proj[i].X
		}
		if proj[i].Y < minY {
			minY = proj[i].Y
		}
		if proj[i].Y > maxY {
			maxY = proj[i].Y
		}
	}

	margin := 60.0 * super
	availW := float64(sw) - 2*margin
	availH := float64(sh) - 2*margin - 40*super
	spanX := maxX - minX
	spanY := maxY - minY
	scale := availW / spanX
	if availH/spanY < scale {
		scale = availH / spanY
	}
	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2
	canvasCX := float64(sw) / 2
	canvasCY := (float64(sh) + 40*super) / 2

	xform := func(p Vec3) (float64, float64) {
		x := canvasCX + (p.X-centerX)*scale
		y := canvasCY - (p.Y-centerY)*scale
		return x, y
	}

	drawAxes(src, xform, camD, super)

	// Screen points and depth.
	snodes := make([]struct{ x, y, f float64 }, len(points))
	for i, p := range proj {
		snodes[i].x, snodes[i].y = xform(p)
		snodes[i].f = p.Z
	}

	// Bonds back-to-front.
	type seg struct{ i, j int; f float64 }
	segs := make([]seg, len(points)-1)
	for i := 0; i < len(points)-1; i++ {
		segs[i] = seg{i, i + 1, (snodes[i].f + snodes[i+1].f) / 2}
	}
	for i := 0; i < len(segs); i++ {
		for j := i + 1; j < len(segs); j++ {
			if segs[j].f < segs[i].f {
				segs[i], segs[j] = segs[j], segs[i]
			}
		}
	}

	for _, s := range segs {
		x1, y1 := snodes[s.i].x, snodes[s.i].y
		x2, y2 := snodes[s.j].x, snodes[s.j].y
		depth := s.f
		w := (1.2 + 2.8*depth) * float64(super)
		alpha := 0.45 + 0.55*depth
		c := color.RGBA{uint8(38 * alpha), uint8(38 * alpha), uint8(38 * alpha), 255}
		line(src, int(x1), int(y1), int(x2), int(y2), int(w+0.5), c)
	}

	// Nodes.
	for i := range snodes {
		x, y, f := snodes[i].x, snodes[i].y, snodes[i].f
		r := (1.5 + 2.5*f) * float64(super)
		alpha := 0.7 + 0.3*f
		c := color.RGBA{0, 0, 0, uint8(255 * alpha)}
		fillCircle(src, int(x), int(y), int(r+0.5), c)
	}

	// Labels on top of geometry.
	for i := range snodes {
		var text string
		switch i {
		case 0:
			text = "p_0"
		case len(snodes) - 1:
			text = "p_N"
		case len(snodes) / 2:
			text = "p_n"
		default:
			continue
		}
		x, y := snodes[i].x, snodes[i].y
		offX, offY := 10.0*super, -10.0*super
		if i == 0 {
			offX, offY = -50*super, -12*super
		} else if i == len(snodes)-1 {
			offX, offY = 14*super, 5*super
		}
		label(src, face, int(x+offX), int(y+offY), text, color.Black)
	}

	// Caption.
	caption := "Seed-chunk mapping: each bond length and direction derives from c_n(s, σ)"
	label(src, smallFace, int(margin), sh-int(10*super), caption, color.RGBA{89, 89, 89, 255})

	// Downsample.
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var r, g, b, a int
			for dy := 0; dy < super; dy++ {
				for dx := 0; dx < super; dx++ {
					c := src.RGBAAt(x*super+dx, y*super+dy)
					r += int(c.R)
					g += int(c.G)
					b += int(c.B)
					a += int(c.A)
				}
			}
			n := super * super
			out.SetRGBA(x, y, color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), uint8(a / n)})
		}
	}
	return out, nil
}

func seedChunk(seed, series string, n int) uint32 {
	if seed == "" {
		// Synthetic deterministic chunks for the illustrative figure.
		h := fnv.New32a()
		fmt.Fprintf(h, "dalle-fold-chunk-%d", n)
		return h.Sum32() % (1 << 24)
	}
	// Derive chunk n from the (seed, series) pair.
	h := fnv.New32a()
	if series != "" {
		fmt.Fprintf(h, "%s-%s-%d", seed, series, n)
	} else {
		fmt.Fprintf(h, "%s-%d", seed, n)
	}
	return h.Sum32() % (1 << 24)
}

func chunkToBond(chunk uint32) (u, theta, phi float64) {
	u = float64((chunk%1000)+1) / 1000.0
	cTheta := chunk & 0xFFF
	cPhi := (chunk >> 12) & 0xFFF
	theta = 2 * math.Pi * float64(cTheta) / 4096.0
	phi = math.Acos(1 - 2*float64(cPhi)/4096.0)
	return
}

func loadFont(size float64) (font.Face, error) {
	ft, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

func label(img *image.RGBA, face font.Face, x, y int, s string, c color.Color) {
	d := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(s)
}

func line(img *image.RGBA, x1, y1, x2, y2, width int, c color.Color) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy
	rad := width / 2
	for {
		fillCircle(img, x1, y1, rad, c)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.Color) {
	rgb := color.RGBAModel.Convert(c).(color.RGBA)
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.SetRGBA(cx+x, cy+y, rgb)
			}
		}
	}
}

func drawAxes(img *image.RGBA, xform func(Vec3) (float64, float64), camD float64, scale int) {
	origin := project(Vec3{0, 0, 0}, camD)
	xAxis := project(Vec3{0.9, 0, 0}, camD)
	yAxis := project(Vec3{0, 0.7, 0}, camD)
	zAxis := project(Vec3{0, 0, 0.9}, camD)

	c := color.RGBA{128, 128, 128, 255}
	ox, oy := xform(origin)
	xx, xy := xform(xAxis)
	yx, yy := xform(yAxis)
	zx, zy := xform(zAxis)

	line(img, int(ox), int(oy), int(xx), int(xy), 2*scale, c)
	line(img, int(ox), int(oy), int(yx), int(yy), 2*scale, c)
	line(img, int(ox), int(oy), int(zx), int(zy), 2*scale, c)
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
