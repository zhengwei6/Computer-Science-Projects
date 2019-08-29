package nchc

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// sX: 0 ~ 900
// sY: -85 ~ +85 / -182 ~ + 182
// sZ: 0 ~ 365

const (
	ovenXmin   = -484
	ovenXmax   = 484
	ovenXshift = 484

	ovenYmin   = -182
	ovenYmax   = 182
	ovenYshift = 182

	ovenZmin   = -182
	ovenZmax   = 182
	ovenZshift = 182

	ovenBottomHeight = 30
)

func normalizeTemp(v, vMin, vMax float64) float64 {
	if vMax == vMin {
		return 0
	}

	return (v - vMin) / (vMax - vMin)
}

func heatColorTwo(v float64) color.NRGBA {
	// R, G, B
	c0 := []uint8{0, 0, 255} // blue
	c1 := []uint8{255, 0, 0} // red
	r := uint8(float64(c0[0])*(1-v) + float64(c1[0])*v)
	g := uint8(float64(c0[1])*(1-v) + float64(c1[1])*v)
	b := uint8(float64(c0[2])*(1-v) + float64(c1[2])*v)
	return color.NRGBA{
		R: r,
		G: g,
		B: b,
		A: 255,
	}
}

func heatColorFive(v float64) color.NRGBA {
	colors := []color.NRGBA{
		color.NRGBA{0, 0, 255, 255},   // blue
		color.NRGBA{0, 255, 255, 255}, // cyan
		color.NRGBA{0, 255, 0, 255},   // green
		color.NRGBA{255, 255, 0, 255}, //yellow
		color.NRGBA{255, 0, 0, 255},   // red
	}
	i := 0
	values := []float64{0.0, 0.25, 0.5, 0.75, 1.0}

	for v > values[i] && i < len(values) {
		i++
	}

	if i == 0 || i == len(values) { // out of range
		return color.NRGBA{
			R: 255,
			G: 255,
			B: 255,
			A: 255,
		}
	}

	c0 := colors[i-1]
	c1 := colors[i]
	v0 := values[i-1]
	v1 := values[i]

	w0 := (v1 - v) / (v1 - v0)
	w1 := (v - v0) / (v1 - v0)

	return color.NRGBA{
		R: uint8(float64(c0.R)*w0 + float64(c1.R)*w1),
		G: uint8(float64(c0.G)*w0 + float64(c1.G)*w1),
		B: uint8(float64(c0.B)*w0 + float64(c1.B)*w1),
		A: uint8(float64(c0.A)*w0 + float64(c1.A)*w1),
	}
}

func evalTempColor(v float64, imgReq *LoadOvenImagesRequest) color.NRGBA {
	heatColor := imgReq.HeatColorMethod
	if v < 0 {
		return color.NRGBA{
			R: 255,
			G: 255,
			B: 255,
			A: 255,
		}
	}
	if v > 1 {
		return color.NRGBA{
			R: 0,
			G: 0,
			B: 0,
			A: 255,
		}
	}
	if heatColor == "two" {
		return heatColorTwo(v)
	}
	return heatColorFive(v)
}

func writeImage(imgName string, img *image.NRGBA) error {
	if len(imgName) < 4 || imgName[len(imgName)-4:len(imgName)] != ".png" {
		imgName += ".png"
	}
	imgName = filepath.FromSlash(imgName)

	dirPath := filepath.Dir(imgName)
	_, err := os.Stat(dirPath)
	if err != nil {
		err := os.MkdirAll(dirPath, 0775)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(imgName)
	if err != nil {
		return err
	}

	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

// YZ plane
// Y -> image x
// Z -> image y
func createImageYZ(imgName string, sensors []*SensorData, imgReq *LoadOvenImagesRequest) error {
	anchor := NewSensorData()
	anchor.X = imgReq.CenterX
	anchor.Y = imgReq.CenterY
	anchor.Z = imgReq.CenterZ

	width := ovenYmax - ovenYmin + 1
	height := ovenZmax - ovenZmin + 1

	// vMin, vMax := getValueRange(sensors)
	vMin := imgReq.MinValue
	vMax := imgReq.MaxValue
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			//compute realted 3D point
			s := NewSensorData()
			s.X = anchor.X
			s.Y = float64(x) - ovenYshift
			s.Z = float64(y) - ovenZshift
			s.Z *= -1 // image is from top to bottom

			// tValue := genValueAlgo0(s, sensors)
			tValue := genValueAlgo1(s, sensors)
			tValue = normalizeTemp(tValue, vMin, vMax)

			c := evalTempColor(tValue, imgReq)

			// check inside
			if s.Y*s.Y+s.Z*s.Z > ovenYshift*ovenZshift {
				c = evalTempColor(-1, imgReq)
			} else if s.Z < (ovenZmin + ovenBottomHeight) {
				c = color.NRGBA{220, 220, 220, 255}
			}

			// draw reference line
			if imgReq.ReferenceLine != "" {
				if int(s.Y) == int(anchor.Y) || int(s.Z) == int(anchor.Z) {
					c = color.NRGBA{0, 0, 0, 255}
				}
			}

			img.Set(x, y, c)
		}
	}

	return writeImage(imgName, img)
}

// XZ plane
// X -> image x
// Z -> image y
func createImageXZ(imgName string, sensors []*SensorData, imgReq *LoadOvenImagesRequest) error {
	anchor := NewSensorData()
	anchor.X = imgReq.CenterX
	anchor.Y = imgReq.CenterY
	anchor.Z = imgReq.CenterZ

	width := ovenXmax - ovenXmin + 1
	height := ovenZmax - ovenZmin + 1

	// vMin, vMax := getValueRange(sensors)
	vMin := imgReq.MinValue
	vMax := imgReq.MaxValue
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			//compute realted 3D point
			s := NewSensorData()
			s.X = float64(x) - ovenXshift
			s.Y = anchor.Y
			s.Z = float64(y) - ovenZshift
			s.Z *= -1 // image is from top to bottom

			// tValue := genValueAlgo0(s, sensors)
			tValue := genValueAlgo1(s, sensors)
			tValue = normalizeTemp(tValue, vMin, vMax)

			c := evalTempColor(tValue, imgReq)

			// check inside
			if s.Z < (ovenZmin + ovenBottomHeight) {
				c = color.NRGBA{220, 220, 220, 255}
			}

			// draw reference line
			if imgReq.ReferenceLine != "" {
				if int(s.X) == int(anchor.X) || int(s.Z) == int(anchor.Z) {
					c = color.NRGBA{0, 0, 0, 255}
				}
			}

			img.Set(x, y, c)
		}
	}

	return writeImage(imgName, img)
}

// XY plane
// X -> image x
// Y -> image y
func createImageXY(imgName string, sensors []*SensorData, imgReq *LoadOvenImagesRequest) error {
	anchor := NewSensorData()
	anchor.X = imgReq.CenterX
	anchor.Y = imgReq.CenterY
	anchor.Z = imgReq.CenterZ

	width := ovenXmax - ovenXmin + 1
	height := ovenYmax - ovenYmin + 1

	// vMin, vMax := getValueRange(sensors)
	vMin := imgReq.MinValue
	vMax := imgReq.MaxValue
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			//compute realted 3D point
			s := NewSensorData()
			s.X = float64(x) - ovenXshift
			s.Y = float64(y) - ovenYshift
			s.Z = anchor.Z
			s.Y *= -1 // image is from top to bottom

			// tValue := genValueAlgo0(s, sensors)
			tValue := genValueAlgo1(s, sensors)
			tValue = normalizeTemp(tValue, vMin, vMax)

			c := evalTempColor(tValue, imgReq)

			// check inside
			if s.Y*s.Y+s.Z*s.Z > ovenYshift*ovenZshift {
				c = evalTempColor(-1, imgReq)
			} else if s.Z < (ovenZmin + ovenBottomHeight) {
				c = color.NRGBA{220, 220, 220, 255}
			}

			// draw reference line
			if imgReq.ReferenceLine != "" {
				if int(s.X) == int(anchor.X) || int(s.Y) == int(anchor.Y) {
					c = color.NRGBA{0, 0, 0, 255}
				}
			}

			img.Set(x, y, c)
		}
	}

	return writeImage(imgName, img)
}

// perspective YZ plane
// Y -> image x
// Z -> image y
func createPerspectiveImageYZ(imgName string, sensors []*SensorData, imgReq *LoadOvenImagesRequest) error {
	width := ovenYmax - ovenYmin + 1
	height := ovenZmax - ovenZmin + 1
	depth := ovenXmax - ovenXmin + 1

	// vMin, vMax := getValueRange(sensors)
	vMin := imgReq.MinValue
	vMax := imgReq.MaxValue
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			tMax := 0.0
			tMin := 1.0
			s := NewSensorData()
			for z := 0; z < depth; z++ {
				//compute realted 3D point
				s.X = float64(z) - ovenXshift
				s.Y = float64(x) - ovenYshift
				s.Z = float64(y) - ovenZshift
				s.Z *= -1 // image is from top to bottom
				tValue := genValueAlgo1(s, sensors)
				tValue = normalizeTemp(tValue, vMin, vMax)
				if tValue > tMax {
					tMax = tValue
				}
				if tValue < tMin {
					tMin = tValue
				}
			}

			// c := evalTempColor(tValue, imgReq)
			c := evalTempColor(tMax, imgReq)
			// c := evalTempColor(tMin, imgReq)

			// check inside
			if s.Y*s.Y+s.Z*s.Z > ovenYshift*ovenZshift {
				c = evalTempColor(-1, imgReq)
			} else if s.Z < (ovenZmin + ovenBottomHeight) {
				c = color.NRGBA{220, 220, 220, 255}
			}

			img.Set(x, y, c)
		}
	}

	return writeImage(imgName, img)
}

// perspective XZ plane
// X -> image x
// Z -> image y
func createPerspectiveImageXZ(imgName string, sensors []*SensorData, imgReq *LoadOvenImagesRequest) error {
	width := ovenXmax - ovenXmin + 1
	height := ovenZmax - ovenZmin + 1
	depth := ovenYmax - ovenYmin + 1

	// vMin, vMax := getValueRange(sensors)
	vMin := imgReq.MinValue
	vMax := imgReq.MaxValue
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			tMax := 0.0
			tMin := 1.0
			s := NewSensorData()
			for z := 0; z < depth; z++ {
				//compute realted 3D point
				s.X = float64(x) - ovenXshift
				s.Y = float64(z) - ovenYshift
				s.Z = float64(y) - ovenZshift
				s.Z *= -1 // image is from top to bottom
				tValue := genValueAlgo1(s, sensors)
				tValue = normalizeTemp(tValue, vMin, vMax)
				if tValue > tMax {
					tMax = tValue
				}
				if tValue < tMin {
					tMin = tValue
				}
			}

			c := evalTempColor(tMax, imgReq)
			// c := evalTempColor(tMin, imgReq)

			// check inside
			if s.Z < (ovenZmin + ovenBottomHeight) {
				c = color.NRGBA{220, 220, 220, 255}
			}

			img.Set(x, y, c)
		}
	}

	return writeImage(imgName, img)
}

// perspective XY plane
// X -> image x
// Y -> image y
func createPerspectiveImageXY(imgName string, sensors []*SensorData, imgReq *LoadOvenImagesRequest) error {
	width := ovenXmax - ovenXmin + 1
	height := ovenYmax - ovenYmin + 1
	depth := ovenZmax - ovenZmin + 1

	// vMin, vMax := getValueRange(sensors)
	vMin := imgReq.MinValue
	vMax := imgReq.MaxValue
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			tMax := 0.0
			tMin := 1.0
			s := NewSensorData()
			for z := 0; z < depth; z++ {
				//compute realted 3D point
				s.X = float64(x) - ovenXshift
				s.Y = float64(y) - ovenYshift
				s.Z = float64(z) - ovenZshift
				s.Y *= -1 // image is from top to bottom
				tValue := genValueAlgo1(s, sensors)
				tValue = normalizeTemp(tValue, vMin, vMax)
				if tValue > tMax {
					tMax = tValue
				}
				if tValue < tMin {
					tMin = tValue
				}
			}

			c := evalTempColor(tMax, imgReq)
			// c := evalTempColor(tMin, imgReq)

			// check inside
			// if s.Y*s.Y+s.Z*s.Z > ovenYshift*ovenZshift {
			// 	c = evalTempColor(-1, imgReq)
			// } else if s.Z < (ovenZmin + ovenBottomHeight) {
			// 	c = color.NRGBA{220, 220, 220, 255}
			// }

			img.Set(x, y, c)
		}
	}

	return writeImage(imgName, img)
}

// XY plane
// X -> image x
// Y -> image y
func createImageXYFromType(imgName string, sensors []*SensorData, imgReq *LoadOvenImagesRequest, sensorType string) error {
	anchor := NewSensorData()
	anchor.X = imgReq.CenterX
	anchor.Y = imgReq.CenterY
	anchor.Z = imgReq.CenterZ

	width := ovenXmax - ovenXmin + 1
	height := ovenYmax - ovenYmin + 1

	// vMin, vMax := getValueRange(sensors)
	vMin := imgReq.MinValue
	vMax := imgReq.MaxValue
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	rackData, rackErr := readRackConfig()

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			//compute realted 3D point
			s := NewSensorData()
			s.X = float64(x) - ovenXshift
			s.Y = float64(y) - ovenYshift
			s.Z = anchor.Z
			s.Y *= -1 // image is from top to bottom

			// tValue := genValueAlgo0(s, sensors)
			tValue := genValueAlgo1FromType(s, sensors, sensorType)
			tValue = normalizeTemp(tValue, vMin, vMax)

			c := evalTempColor(tValue, imgReq)

			// check inside
			if s.Y*s.Y+s.Z*s.Z > ovenYshift*ovenZshift {
				c = evalTempColor(-1, imgReq)
			} else if s.Z < (ovenZmin + ovenBottomHeight) {
				c = color.NRGBA{220, 220, 220, 255}
			}

			// draw reference line
			if imgReq.ReferenceLine != "" {
				if int(s.X) == int(anchor.X) || int(s.Y) == int(anchor.Y) {
					c = color.NRGBA{0, 0, 0, 255}
				}
			}

			// draw aidc sensor location
			if sensorType == "aidc" && rackErr == nil {
				xInside := false
				xOnline := false
				yInside := false
				yOnline := false
				for _, row := range rackData.RowVals {
					if int(s.Y) <= int(row+mCubeSizeY) && int(s.Y) >= int(row-mCubeSizeY) {
						yInside = true
						if int(s.Y) == int(row+mCubeSizeY) || int(s.Y) == int(row-mCubeSizeY) {
							yOnline = true
						}
						break
					}
				}

				for _, col := range rackData.ColVals {
					if int(s.X) <= int(col+mCubeSizeX) && int(s.X) >= int(col-mCubeSizeX) {
						xInside = true
						if int(s.X) == int(col+mCubeSizeX) || int(s.X) == int(col-mCubeSizeX) {
							xOnline = true
						}
						break
					}
				}
				if (xOnline && yInside) || (yOnline && xInside) {
					c = color.NRGBA{0, 0, 0, 255}
				}
			}

			img.Set(x, y, c)
		}
	}

	return writeImage(imgName, img)
}
